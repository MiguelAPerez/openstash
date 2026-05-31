package spec

import "gopkg.in/yaml.v3"

func unmarshalYAML(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}
