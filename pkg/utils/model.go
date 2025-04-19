package utils

type VersionResponse struct {
	Version string `json:"version"`
}

type Error struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
	Origin  string `json:"origin"`
}

type Parameter struct {
	Type  string `query:"type" optional:"true"`
	Value string `query:"value" optional:"true"`
}
