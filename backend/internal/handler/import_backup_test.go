package handler_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

// TestImportBackup_DurationIsSeconds verifies the PlaybackReporting TSV import
// stores PlayDuration verbatim as seconds. The plugin already reports seconds,
// so dividing by 10,000,000 (ticks) would wipe out the watch time.
func TestImportBackup_DurationIsSeconds(t *testing.T) {
	db := testutil.NewDB(t)

	// TSV: DateCreated \t UserId \t ItemId \t ItemType \t ItemName \t Method \t Client \t Device \t PlayDuration
	tsv := "DateCreated\tUserId\tItemId\tItemType\tItemName\tMethod\tClient\tDevice\tPlayDuration\n" +
		"2026-07-12\tuser-1\titem-1\tMovie\tInception\tDirectPlay\tWeb\tChrome\t600\n"

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", "backup.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(tsv)); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	gin.SetMode(gin.TestMode)
	h := handler.NewTasksApiHandler(nil, repository.New(db), nil)
	r := gin.New()
	r.POST("/import", h.ImportBackup)

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	var got models.JFPlaybackActivity
	if err := db.Where(`"Id" = ?`, "import-user-1-item-1-2026-07-12").First(&got).Error; err != nil {
		t.Fatalf("imported row not found: %v", err)
	}
	if got.PlaybackDuration == nil || *got.PlaybackDuration != 600 {
		t.Errorf("PlaybackDuration = %v, want 600 seconds (must not divide by ticks)", got.PlaybackDuration)
	}
	if got.Source != "import" {
		t.Errorf("Source = %q, want import", got.Source)
	}
	if !got.Imported {
		t.Error("Imported flag should be true")
	}
}
