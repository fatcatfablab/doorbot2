package wsreader

type wsMsg struct {
	Event string `json:"event"`
	Data  wsData `json:"data"`
}

type wsData struct {
	Source wsSource `json:"_source"`
}

type wsSource struct {
	Actor wsActor `json:"actor"`
	Event wsEvent `json:"event"`
}

type wsActor struct {
	Id          string `json:"id"`
	DisplayName string `json:"display_name"`
	AlternateId string `json:"alternate_id"`
}

type wsEvent struct {
	Type   string `json:"type"`
	Result string `json:"result"`
}
