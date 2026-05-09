package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed crds/application-crd.yaml
var applicationCRD []byte

//go:embed crds/applicationset-crd.yaml
var applicationsetCRD []byte

//go:embed crds/appproject-crd.yaml
var appprojectCRD []byte

func parseCRDs() ([]json.RawMessage, error) {
	yamls := [][]byte{applicationCRD, applicationsetCRD, appprojectCRD}
	out := make([]json.RawMessage, len(yamls))
	for i, y := range yamls {
		raw, err := yamlToJSON(y)
		if err != nil {
			return nil, fmt.Errorf("crd[%d]: %w", i, err)
		}
		out[i] = raw
	}
	return out, nil
}
