package discordgox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
)

// GuildStickerCreate uploads a new sticker via multipart form. discordgo has no
// typed helper for guild sticker creation.
func (c *Client) GuildStickerCreate(guildID, name, description, tags, filename string, fileData []byte) (*discordgo.Sticker, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("name", name)
	writer.WriteField("description", description)
	writer.WriteField("tags", tags)

	part, err := createFormFileWithContentType(writer, "file", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	part.Write(fileData)
	writer.Close()

	endpoint := discordgo.EndpointGuildStickers(guildID)
	resp, err := c.RequestRaw("POST", endpoint, writer.FormDataContentType(), body.Bytes(), endpoint, 0)
	if err != nil {
		return nil, err
	}

	var sticker discordgo.Sticker
	if err := json.Unmarshal(resp, &sticker); err != nil {
		return nil, err
	}
	return &sticker, nil
}

// GuildSticker fetches a single guild sticker.
func (c *Client) GuildSticker(guildID, stickerID string) (*discordgo.Sticker, error) {
	endpoint := discordgo.EndpointGuildSticker(guildID, stickerID)
	resp, err := c.RequestWithBucketID("GET", endpoint, nil, endpoint)
	if err != nil {
		return nil, err
	}

	var sticker discordgo.Sticker
	if err := json.Unmarshal(resp, &sticker); err != nil {
		return nil, err
	}
	return &sticker, nil
}

// GuildStickerEdit updates a guild sticker's metadata.
func (c *Client) GuildStickerEdit(guildID, stickerID, name, description, tags string) error {
	data := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Tags        string `json:"tags"`
	}{Name: name, Description: description, Tags: tags}

	endpoint := discordgo.EndpointGuildSticker(guildID, stickerID)
	_, err := c.RequestWithBucketID("PATCH", endpoint, data, endpoint)
	return err
}

// GuildStickerDelete removes a guild sticker.
func (c *Client) GuildStickerDelete(guildID, stickerID string) error {
	endpoint := discordgo.EndpointGuildSticker(guildID, stickerID)
	_, err := c.RequestWithBucketID("DELETE", endpoint, nil, endpoint)
	return err
}

func createFormFileWithContentType(w *multipart.Writer, fieldname, filename string) (io.Writer, error) {
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldname, filename))
	h.Set("Content-Type", contentType)

	return w.CreatePart(h)
}
