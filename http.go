package gotils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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

// ErrorResponse ...
// Note: this is compatible with firebase errors too
type ErrorResponse struct {
	Error *BasicResponse `json:"error"`
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
			handleErr(w, err)
		}
	}
}

func handleErr(w http.ResponseWriter, err error) {
	// DumpError(err)
	if errors.Is(err, ErrNotFound) {
		WriteError(w, http.StatusNotFound, err)
		return
	}
	if pf != nil {
		// send to user defined output
		pf.Printf("%v", err)
	}
	if loggable != nil {
		// send to user defined output
		loggable.Logf(context.Background(), "error", "%v", err)
	}
	var ue UserError
	if errors.As(err, &ue) {
		// fmt.Println("user error")
		WriteError(w, http.StatusBadRequest, errors.New(ue.UserError()))
		return
	}
	var he HTTPError
	if errors.As(err, &he) {
		// fmt.Println("http error", he.Code())
		WriteError(w, he.Code(), he)
		return
	}
	// default:
	WriteError(w, http.StatusInternalServerError, err)
}

type ObjectNamer interface {
	ObjectName() string
}

type ObjectHandlerFunc func(w http.ResponseWriter, r *http.Request) (ObjectNamer, error)

// ErrorHandler a generic error handler that will respond with a generic error response
// Set a logger/printer with gotils.SetPrintfer() to have this log to your logger
func ObjectHandler(h ObjectHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := h(w, r)
		if err != nil {
			handleErr(w, err)
			return
		}
		if v == nil {
			handleErr(w, ErrNotFound)
			return
		}
		// respond with object
		WriteObject(w, 200, map[string]interface{}{v.ObjectName(): v})
	}
}

func WriteError(w http.ResponseWriter, code int, err error) {
	switch err.(type) {
	case *DetailedError:
		WriteObject(w, code, map[string]interface{}{"error": err})
	default:
		WriteObject(w, code, map[string]interface{}{"error": map[string]interface{}{"message": err.Error(), "status": code}})
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
		return nil, NewHTTPError(fmt.Sprintf("Error response %v: %v", resp.StatusCode, string(bodyBytes)), resp.StatusCode)
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
	if strings.HasPrefix(url, "ipfs://") {
		path := strings.TrimPrefix(url, "ipfs://")
		// url = fmt.Sprintf("https://cloudflare-ipfs.com/ipfs/%v", path)
		// cloudflare giving me captcha response!
		url = fmt.Sprintf("https://ipfs.io/ipfs/%v", path)
		fmt.Println("URL:", url)
	}
	resp, err := http.Get(url)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	defer resp.Body.Close()

	err = checkError(resp)
	if err != nil {
		return C(context.Background()).Error(err)
	}

	err = ParseJSONReader(resp.Body, t)
	if err != nil {
		return C(context.Background()).Errorf("couldn't parse response: %v", err)
	}
	return nil
}

type RequestOptions struct {
	Headers map[string]string
}

// GetJSON performs a get request and then parses the result into t
func GetJSONOpts(url string, t interface{}, opts *RequestOptions) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	// req.Header = http.Header{
	// "Host":          []string{"www.host.com"},
	// "Content-Type":  []string{"application/json"},
	// "Authorization": []string{"Bearer Token"},
	// }
	if opts == nil {
		opts = &RequestOptions{
			Headers: map[string]string{},
		}
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	defer resp.Body.Close()

	err = checkError(resp)
	if err != nil {
		return C(context.Background()).Error(err)
	}

	err = ParseJSONReader(resp.Body, t)
	if err != nil {
		return C(context.Background()).Errorf("couldn't parse response: %v", err)
	}
	return nil
}

// PostJSON performs a post request with tin as the body then parses the response into tout. tin and tout can be the same object.
func PostJSON(url string, tin, tout interface{}) error {
	jsonValue, err := json.Marshal(tin)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return C(context.Background()).Error(err)
	}
	defer resp.Body.Close()

	err = checkError(resp)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	if tout != nil {
		err = ParseJSONReader(resp.Body, tout)
		if err != nil {
			return fmt.Errorf("couldn't parse response: %v", err)
		}
	}
	return nil
}

// GetJSON performs a get request and then parses the result into t
func PostJSONOpts(url string, tin, tout interface{}, opts *RequestOptions) error {
	jsonValue, err := json.Marshal(tin)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return C(context.Background()).Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if opts == nil {
		opts = &RequestOptions{
			Headers: map[string]string{},
		}
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	defer resp.Body.Close()

	err = checkError(resp)
	if err != nil {
		return C(context.Background()).Error(err)
	}

	err = ParseJSONReader(resp.Body, tout)
	if err != nil {
		return C(context.Background()).Errorf("couldn't parse response: %v", err)
	}
	return nil
}

// PatchJSON performs a PATCH request with tin as the body then parses the response into tout. tin and tout can be the same object.
func PatchJSON(url string, tin, tout interface{}) error {

	jsonValue, err := json.Marshal(tin)
	if err != nil {
		return C(context.Background()).Error(err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return C(context.Background()).Error(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return C(context.Background()).Error(err)
	}
	// resp, err := http.Patch(url, "application/json", bytes.NewBuffer(jsonValue))
	// if err != nil {
	// 	return err
	// }
	defer resp.Body.Close()

	err = checkError(resp)
	if err != nil {
		return err
	}
	if tout != nil {
		err = ParseJSONReader(resp.Body, tout)
		if err != nil {
			return fmt.Errorf("couldn't parse response: %v", err)
		}
	}
	return nil
}

func checkError(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// attempt to parse JSON
		er := &ErrorResponse{}
		err2 := ParseJSONBytes(bodyBytes, er)
		if err2 == nil {
			if er.Error != nil && er.Error.Message != "" {
				return NewHTTPError(er.Error.Message, resp.StatusCode)
			}
		}
		// couldn't parse or no message, so just send regular error
		return NewHTTPError(fmt.Sprintf("Error %v: %v", resp.StatusCode, string(bodyBytes)), resp.StatusCode)
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
