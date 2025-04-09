package app

var (
	version        string
	go_version     string
	build_time     string
	commit_hash    string
	git_branch     string
	git_tree_state string
)

// BuildInfo describes the compile time information.
// type BuildInfo struct {
// 	Version      string `json:"version,omitempty"`
// 	GoVersion    string `json:"go_version,omitempty"`
// 	BuildTime    string `json:"build_time,omitempty"`
// 	CommitHash   string `json:"commit_hash,omitempty"`
// 	GitBranch    string `json:"git_branch,omitempty"`
// 	GitTreeState string `json:"git_tree_state,omitempty"`
// }

// GetVersion returns the version of the binary
func Version() string {
	return version
}

// Info returns build info
// func BuildInfo() BuildInfo {
// 	v := BuildInfo{
// 		Version:      version,
// 		GoVersion:    go_version,
// 		BuildTime:    build_time,
// 		CommitHash:   commit_hash,
// 		GitBranch:    git_branch,
// 		GitTreeState: git_tree_state,
// 	}

// 	return v
// }
