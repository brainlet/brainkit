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

func (FsReadResp) BusTopic() string { return "fs.read.result" }

type FsListResp struct {
	ResultMeta
	Files []FsFileInfo `json:"files"`
}

func (FsListResp) BusTopic() string { return "fs.list.result" }

type FsStatResp struct {
	ResultMeta
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
}

func (FsStatResp) BusTopic() string { return "fs.stat.result" }

type FsWriteResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (FsWriteResp) BusTopic() string { return "fs.write.result" }

type FsDeleteResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (FsDeleteResp) BusTopic() string { return "fs.delete.result" }

type FsMkdirResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (FsMkdirResp) BusTopic() string { return "fs.mkdir.result" }

// ── Shared types ──

type FsFileInfo struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}
