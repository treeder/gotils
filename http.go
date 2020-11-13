package gotils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

var (
	pf Printfer
)

// SetPrintfer to let this library print errors to your logging library
func SetPrintfer(p Printfer) {
	pf = p
}

// Port will check env var for PORT (common on cloud services) and use that, otherwise
// it will return the provided port.
func Port(port int) int {
	// portS := strconv.Itoa(port)
	portS := os.Getenv("PORT")
	if portS != "" {
		port, _ = strconv.Atoi(portS)
	}
	return port
}

type BasicResponse struct {
	Message string `json:"message"`
}

// TODO: need a UserError type that can have a message for a user and the raw error message for logging
// maybe a NewUserError("some user message", rawError to wrap and user for logging)

type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ErrorHandler a generic error handler that will respond with a generic error response
// Set a logger/printer with gotils.SetPrintfer() to have this log to your logger
func ErrorHandler(h ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			if pf != nil {
				pf.Printf("%v", err)
			}
			if errors.Is(err, ErrNotFound) {
				WriteError(w, http.StatusNotFound, err)
			}
			switch e := err.(type) {
			case *HttpError:
				// gcputils.Error().Printf("%v", err)
				WriteError(w, e.Code(), e)
			default:
				// gcputils.Error().Printf("%v", err) // to cloud logging
				WriteError(w, http.StatusInternalServerError, e)
			}
		}
	}
}

func WriteError(w http.ResponseWriter, code int, err error) {
	switch err.(type) {
	case *DetailedError:
		WriteObject(w, code, map[string]interface{}{"error": err})
	default:
		WriteObject(w, code, map[string]interface{}{"error": map[string]interface{}{"message": err.Error()}})
	}
}

func WriteMessage(w http.ResponseWriter, code int, msg string) {
	WriteObject(w, 200, map[string]interface{}{
		"message": msg,
	})
}

func WriteObject(w http.ResponseWriter, code int, obj interface{}) {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		log.Printf("ERROR: couldn't marshal JSON in WriteObject: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write([]byte(jsonValue))
	if err != nil {
		log.Printf("ERROR: couldn't write error response: %v", err)
	}
}

func ParseJSON(w http.ResponseWriter, r *http.Request, t interface{}) error {
	err := ParseJSONReader(r.Body, t)
	if err != nil {
		return err
	}
	return nil
}

func ParseJSONBytes(b []byte, t interface{}) error {
	return json.Unmarshal(b, t)
}

func ParseJSONReader(r io.Reader, t interface{}) error {
	decoder := json.NewDecoder(r)
	err := decoder.Decode(t)
	return err
}

func BytesToJSON(bs []byte) (string, error) {
	return ToJSON(string(bs))
}

func ToJSON(v interface{}) (string, error) {
	jsonValue, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonValue), nil
}

func GetBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if err != nil {
			return nil, err
		}
		return nil, NewHttpError(fmt.Sprintf("Error response %v: %v", resp.StatusCode, string(bodyBytes)), resp.StatusCode)
	}
	return bodyBytes, nil
}

func GetString(url string) (string, error) {
	b, err := GetBytes(url)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetJSON performs a get request and then parses the result into t
func GetJSON(url string, t interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return NewHttpError(fmt.Sprintf("Error response %v: %v", resp.StatusCode, string(bodyBytes)), resp.StatusCode)
	}

	err = ParseJSONReader(resp.Body, t)
	if err != nil {
		return fmt.Errorf("couldn't parse response: %v", err)
	}
	return nil
}

// PostJSON performs a post request with tin as the body then parses the response into tout. tin and tout can be the same object.
func PostJSON(url string, tin, tout interface{}) error {
	jsonValue, err := json.Marshal(tin)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return NewHttpError(fmt.Sprintf("Error response %v: %v", resp.StatusCode, string(bodyBytes)), resp.StatusCode)
	}
	err = ParseJSONReader(resp.Body, tout)
	if err != nil {
		return fmt.Errorf("couldn't parse response: %v", err)
	}
	return nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) (*os.File, error) {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := ioutil.TempFile("", filepath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return out, err
}
