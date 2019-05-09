package gotils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	ipfsapi "github.com/ipfs/go-ipfs-api"
)

var (
	ipfs   *ipfsapi.Shell
	ipfsUp bool
)

func init() {
	ipfs = ipfsapi.NewShell("localhost:5001")
	ipfsUp = ipfs.IsUp() // could recheck this every few minutes
	if !ipfsUp {
		fmt.Println("ipfs daemon not running, using infura")
	}
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

func UploadFileToIPFS(ctx context.Context, filePath string) (string, error) {
	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	// writer := multipart.NewWriter(body)
	// part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	io.Copy(body, file)
	// writer.Close()

	return UploadBytesToIPFS(ctx, body.Bytes())
}

func UploadObjectToIPFS(ctx context.Context, data interface{}) (string, error) {
	jsonValue, err := json.Marshal(data)
	if err != nil {
		L(ctx).Error("couldn't marshal JSON in writeJSON! This error is not handled", zap.Error(err))
		return "", err
	}

	return UploadBytesToIPFS(ctx, jsonValue)
}

func UploadBytesToIPFS(ctx context.Context, data []byte) (string, error) {
	// try local first
	if ipfsUp {
		buf := bytes.NewBuffer(data)
		cid, err := ipfs.Add(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			// os.Exit(1)
			return "", err
		}
		fmt.Printf("added %s\n", cid)
		return cid, nil
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "somefile")
	part.Write(data)
	writer.Close()

	return postToInfura(ctx, writer, body)
}

type InfuraIPFSResponse struct {
	// {
	// 	"Name": "sample-result.json",
	// 	"Hash": "QmSTkR1kkqMuGEeBS49dxVJjgHRMH6cUYa7D3tcHDQ3ea3",
	// 	"Size": "2120"
	// }
	Name string
	Hash string
	Size string
}

func postToInfura(ctx context.Context, writer *multipart.Writer, body io.Reader) (string, error) {
	r, _ := http.NewRequest("POST", "https://ipfs.infura.io:5001/api/v0/add?pin=true", body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(r)
	var bodyContent []byte
	if err != nil {
		return "", err
	} else {
		// fmt.Println(resp.StatusCode)
		// fmt.Println(resp.Header)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			L(ctx).Error("Got error back from infura", zap.Error(err))
			return "", fmt.Errorf("Got status code %v back from IPFS server", resp.StatusCode)
		}
		bodyContent, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			L(ctx).Error("error unmarshaling message from infura", zap.Error(err))
			return "", err
		}
		defer resp.Body.Close()
		fmt.Println(string(bodyContent))
	}
	ipfsResp := &InfuraIPFSResponse{}
	err = json.Unmarshal(bodyContent, ipfsResp)
	if err != nil {
		L(ctx).Error("error unmarshaling message from infura", zap.Error(err))
		return "", err
	}
	cid := ipfsResp.Hash
	// fire off a couple gets to ipfs.io and cloudflare so they cache it quicker
	go GetString("https://ipfs.io/ipfs/" + cid)
	go GetString("https://cloudflare-ipfs.com/ipfs/" + cid)
	return cid, nil
}

func GetBytesFromIPFS(ctx context.Context, cid string) ([]byte, error) {
	var stateBytes []byte
	var err error
	if ipfsUp {
		// todo: need to add a timeout here, this can take a while if file is not local
		// todo: the ipfs lib might need to chanage to support, should bre able to pass a context into all these methods
		reader, err := ipfs.Cat(cid)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		stateBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		// try http gateway
		stateBytes, err = GetBytes("https://cloudflare-ipfs.com/ipfs/" + cid)
		if err != nil {
			return nil, err
		}
	}
	return stateBytes, nil
}
