package gotils

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

func ParseJSONFile(filename string, t interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
	   return err
	}
	return ParseJSONReader(f, t)
}