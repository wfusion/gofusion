package cases

var (
	// apolloProperties in default "application" namespace
	apolloProperties = map[string]string{
		"test.timeout": "50s",
		"test.user":    "gopher-prop",
	}

	apolloYamlNamespace = "app.yaml"
	apolloJsonNamespace = "app.json"
	apolloTxtNamespace  = "test.txt"
	apolloTxtData       = "This is a plain text configuration."
)
