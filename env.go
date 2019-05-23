package gotils

import (
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/compute/metadata"
)

// GetEnvVar def is default, leave blank to fatal if not found
// checks in env, then GCE metadata
func GetEnvVar(name, def string) string {
	var err error
	e := os.Getenv(name)
	if e != "" {
		return e
	}
	// check if a metadata.json file exists, this is the file downloaded from google "REST equivalent" in metadata section
	// if len(metaFileItems) > 0 {
	// 	// see if we loaded it from a file
	// 	for _, kv := range metaFileItems {
	// 		fmt.Println(kv.Key, kv.Value)
	// 		if kv.Key == name {
	// 			return kv.Value
	// 		}
	// 	}
	// }
	if metadata.OnGCE() {
		e, err = metadata.ProjectAttributeValue(name)
		if err == nil {
			fmt.Println("GOT META", e)
			return e
		}
		log.Println("error on metadata.ProjectAttributeValue", err)
	}
	if def != "" {
		return def
	}
	// log.Info().Str(name, tgApiKey).Msg("Got from metadata server")
	log.Fatalf("NO %v", name)
	return e
}
