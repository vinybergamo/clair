package apps

var (
	DefaultProperties = map[string]string{
		"deploy-source":          "",
		"deploy-source-metadata": "",
	}

	GlobalProperties = map[string]bool{
		"deploy-source":          true,
		"deploy-source-metadata": true,
	}
)
