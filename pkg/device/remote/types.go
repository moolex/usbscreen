package remote

type EmptyResponse struct {
}

type SetRotateRequest struct {
	Landscape bool
	Invert    bool
}

type DrawBitmapRequest struct {
	PosX  uint16
	PosY  uint16
	Image []byte
}
