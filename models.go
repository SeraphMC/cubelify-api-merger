package main

import "github.com/charmbracelet/bubbles/textinput"

type APIConfigs map[string]APIConfig

type APIConfig struct {
	URL           string                 `json:"url"`
	Querystring   map[string]interface{} `json:"querystring"`
	RequestParams map[string]string      `json:"request_params"`
}

type ClipboardMsg struct {
	Success bool
	Err     error
}

type MenuModel struct {
	Cursor       int
	Choices      []string
	URLCopied    bool
	ClipboardErr string
}

type SelectionModel struct {
	Cursor  int
	Items   []string
	Mode    string
	Deleted string
}

type FormModel struct {
	Inputs  []textinput.Model
	Focus   int
	ErrMsg  string
	Success bool
}
