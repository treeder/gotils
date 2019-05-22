package gotils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type BasicResponse struct {
	Message string `json:"message"`
}

type DetailedError struct {
	Message string `json:"message"`
	Details string `json:"details"`
}

func (e *DetailedError) Error() string {
	return e.Message
}

func WriteError(w http.ResponseWriter, code int, err error) {
	switch t := err.(type) {
	case *DetailedError:
		fmt.Println("ModelMissingError", t)
	default:
		bodyMap := map[string]interface{}{"error": map[string]interface{}{"message": err.Error()}}
	}
	WriteObject(w, code, bodyMap)
}

func WriteMessage(w http.ResponseWriter, code int, msg string) {
	WriteObject(w, 200, map[string]interface{}{
		"message": msg,
	})
}

func WriteObject(w http.ResponseWriter, code int, obj interface{}) {
	jsonValue, _ := json.Marshal(obj)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := w.Write([]byte(jsonValue))
	if err != nil {
		log.Printf("ERROR: couldn't write error response: %v", err)
	}
}

func ParseJSON(w http.ResponseWriter, r *http.Request, t interface{}) error {
	err := ParseJSONReader(r.Body, t)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request body, bad JSON: %v", err))
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
		return nil, (fmt.Errorf("Error response %v: %v", resp.StatusCode, string(bodyBytes)))
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
		return (fmt.Errorf("Error response %v: %v", resp.StatusCode, string(bodyBytes)))
	}

	err = ParseJSONReader(resp.Body, t)
	if err != nil {
		return fmt.Errorf("couldn't parse response: %v", err)
	}
	return nil
}

func PostJSON(url string, t interface{}) error {
	jsonValue, err := json.Marshal(t)
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
		return (fmt.Errorf("Error response %v: %v", resp.StatusCode, string(bodyBytes)))
	}
	err = ParseJSONReader(resp.Body, t)
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
