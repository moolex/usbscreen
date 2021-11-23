package remote

import (
	"bytes"
	"image"
	"image/png"
	"net/rpc"

	"usbscreen/pkg/proto"
)

func New(addr string) (proto.Control, error) {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Client{rpc: client}, nil
}

type Client struct {
	rpc *rpc.Client
}

func (c *Client) Startup() error {
	return c.rpc.Call("Service.Command", "startup", nil)
}

func (c *Client) Shutdown() error {
	return c.rpc.Call("Service.Command", "shutdown", nil)
}

func (c *Client) Restart() error {
	return c.rpc.Call("Service.Command", "restart", nil)
}

func (c *Client) SetLight(light uint8) error {
	return c.rpc.Call("Service.SetLight", light, nil)
}

func (c *Client) SetMirror(mirror bool) error {
	return c.rpc.Call("Service.SetMirror", mirror, nil)
}

func (c *Client) SetRotate(landscape bool, invert bool) error {
	return c.rpc.Call("Service.SetRotate", SetRotateRequest{
		Landscape: landscape,
		Invert:    invert,
	}, nil)
}

func (c *Client) DrawBitmap(posX uint16, posY uint16, image image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image); err != nil {
		return err
	}

	return c.rpc.Call("Service.DrawBitmap", &DrawBitmapRequest{
		PosX:  posX,
		PosY:  posY,
		Image: buf.Bytes(),
	}, nil)
}
