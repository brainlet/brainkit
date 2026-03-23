package messages

// ── Requests ──

type FsReadMsg struct {
	Path string `json:"path"`
}

func (FsReadMsg) BusTopic() string { return "fs.read" }

type FsWriteMsg struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

func (FsWriteMsg) BusTopic() string { return "fs.write" }

type FsListMsg struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

func (FsListMsg) BusTopic() string { return "fs.list" }

type FsStatMsg struct {
	Path string `json:"path"`
}

func (FsStatMsg) BusTopic() string { return "fs.stat" }

type FsDeleteMsg struct {
	Path string `json:"path"`
}

func (FsDeleteMsg) BusTopic() string { return "fs.delete" }

type FsMkdirMsg struct {
	Path string `json:"path"`
}

func (FsMkdirMsg) BusTopic() string { return "fs.mkdir" }

// ── Responses ──

type FsReadResp struct {
	ResultMeta
	Data string `json:"data"`
}


type FsListResp struct {
	ResultMeta
	Files []FsFileInfo `json:"files"`
}


type FsStatResp struct {
	ResultMeta
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
}


type FsWriteResp struct {
	ResultMeta
	OK bool `json:"ok"`
}


type FsDeleteResp struct {
	ResultMeta
	OK bool `json:"ok"`
}


type FsMkdirResp struct {
	ResultMeta
	OK bool `json:"ok"`
}


// ── Shared types ──

type FsFileInfo struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}
