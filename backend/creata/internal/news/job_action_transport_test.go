package news

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestJobActionConsumesJSONBodyBeforeLargeCloseResponse(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	job := newJob("draft", Doc{"evidence": strings.Repeat("Retained source evidence. ", 10000)})
	if e := a.store.Write(context.Background(), func(tx *Tx) error { return tx.Put(context.Background(), "job", job) }); e != nil {
		t.Fatal(e)
	}
	invalid, _ := http.NewRequest("POST", a.server.URL+base+"/admin/jobs/"+str(job["id"])+"/cancel", bytes.NewBufferString("{"))
	invalid.Close = true
	invalid.Header.Set("Content-Type", "application/json")
	invalid.Header.Set("X-CSRF-Token", a.csrf)
	rejected, e := a.client.Do(invalid)
	if e != nil {
		t.Fatal(e)
	}
	io.Copy(io.Discard, rejected.Body)
	rejected.Body.Close()
	if rejected.StatusCode != 400 {
		t.Fatalf("invalid action JSON was not consumed: %d", rejected.StatusCode)
	}
	request, _ := http.NewRequest("POST", a.server.URL+base+"/admin/jobs/"+str(job["id"])+"/cancel", bytes.NewBufferString("{}"))
	request.Close = true
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", a.csrf)
	response, e := a.client.Do(request)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(response.Body)
	if e != nil {
		t.Fatal(e)
	}
	var result Doc
	if e = json.Unmarshal(raw, &result); e != nil {
		t.Fatal(e)
	}
	if str(result["status"]) != "cancelled" {
		t.Fatalf("unexpected response status: %d", response.StatusCode)
	}
}
