package provider

import "fmt"

func ForName(name string) (Provider, error) {
	switch name {
	case "aws":
		return &AWSProvider{}, nil
	case "gcp":
		return &GCPProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider %q: supported: aws, gcp", name)
	}
}
