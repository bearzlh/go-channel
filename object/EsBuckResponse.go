package object

type EsBuckResponse struct {
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			Status int32   `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"index"`
	} `json:"items"`
}
