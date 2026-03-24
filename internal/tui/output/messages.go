package output

type OutputLineMsg struct {
	Text     string
	IsStderr bool
}

type ExecutionDoneMsg struct {
	LogPath string
	Err     error
}

type CopiedHeaderFlashMsg struct{}
type CopiedFooterFlashMsg struct{}
type CopyFlashExpiredMsg struct{}
type CopiedRegionFlashMsg struct{} // clears header/footer background highlight
type SelectionCompleteMsg struct{} // signals the app to extract and copy selected text
