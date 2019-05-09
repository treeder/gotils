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

func writeError(w http.ResponseWriter, code int, err error) {
	bodyMap := map[string]interface{}{"error": map[string]interface{}{"message": err.Error()}}
	writeJSON(w, code, bodyMap)
}

func writeJSON(w http.ResponseWriter, code int, obj map[string]interface{}) {
	writeObject(w, code, obj)
}

func writeMessage(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, 200, map[string]interface{}{
		"message": msg,
	})
}

func writeObject(w http.ResponseWriter, code int, obj interface{}) {
	jsonValue, _ := json.Marshal(obj)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := w.Write([]byte(jsonValue))
	if err != nil {
		log.Printf("ERROR: couldn't write error response: %v", err)
	}
}

func parseJSON(w http.ResponseWriter, r *http.Request, t interface{}) error {
	err := parseJSONReader(r.Body, t)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body, bad JSON: %v", err))
		return err
	}
	return nil
}

func parseJSONBytes(b []byte, t interface{}) error {
	return json.Unmarshal(b, t)
}

func parseJSONReader(r io.Reader, t interface{}) error {
	decoder := json.NewDecoder(r)
	err := decoder.Decode(t)
	return err
}

func bytesToJSON(bs []byte) (string, error) {
	return toJSON(string(bs))
}

func toJSON(v interface{}) (string, error) {
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

	err = parseJSONReader(resp.Body, t)
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
	err = parseJSONReader(resp.Body, t)
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
