//go:build loadtest

// Uji beban war slot terhadap API yang sedang berjalan.
//
//	docker compose up -d --build
//	docker compose exec -T postgres psql -U postgres -d bansos_backend < tests/loadtest/seed_users.sql
//	go test -tags loadtest -v -count=1 ./tests/loadtest/
//
// Variabel opsional: BANSOS_API_URL (bawaan http://localhost:8081/api/v1),
// WAR_USERS (bawaan 300), WAR_QUOTA (bawaan 10), WAR_MODULE_ID (bawaan 1).
package loadtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"bansos-backend/app/model"
	"bansos-backend/middleware"
)

const seedUserBase = 900000

type envelope struct {
	Success bool            `json:"success"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Meta    struct {
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	} `json:"meta"`
}

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

func token(id int, role string) string {
	return middleware.StubToken(model.AuthUser{UserID: id, Name: "uji", Email: "uji@bansos.test", Role: role})
}

func (c *client) do(method, path, tok string, body any) (int, envelope, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return 0, envelope{}, err
		}
	}
	req, err := http.NewRequest(method, c.base+path, &buf)
	if err != nil {
		return 0, envelope{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, envelope{}, err
	}
	defer resp.Body.Close()

	var env envelope
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&env)
	}
	return resp.StatusCode, env, nil
}

func (c *client) must(method, path, tok string, body any, want int, out any) {
	c.t.Helper()
	status, env, err := c.do(method, path, tok, body)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	if status != want {
		c.t.Fatalf("%s %s: status %d (%s %s), want %d", method, path, status, env.Code, env.Message, want)
	}
	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			c.t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
}

func (c *client) claimsByStatus(batchID int, status, tok string) []model.Claim {
	c.t.Helper()
	var all []model.Claim
	cursor := ""
	for {
		path := fmt.Sprintf("/batches/%d/claims?status=%s&limit=100&cursor=%s", batchID, status, cursor)
		code, env, err := c.do(http.MethodGet, path, tok, nil)
		if err != nil || code != http.StatusOK {
			c.t.Fatalf("GET %s: %v status %d", path, err, code)
		}
		var page []model.Claim
		if err := json.Unmarshal(env.Data, &page); err != nil {
			c.t.Fatalf("decode klaim: %v", err)
		}
		all = append(all, page...)
		if !env.Meta.HasMore {
			return all
		}
		cursor = env.Meta.NextCursor
	}
}

func TestWarSlotConcurrent(t *testing.T) {
	base := os.Getenv("BANSOS_API_URL")
	if base == "" {
		base = "http://localhost:8081/api/v1"
	}
	users := envInt("WAR_USERS", 300)
	quota := envInt("WAR_QUOTA", 10)
	moduleID := envInt("WAR_MODULE_ID", 1)
	if users > 500 {
		t.Fatalf("WAR_USERS maksimal 500 sesuai seed_users.sql")
	}

	c := &client{t: t, base: base, http: &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 256},
	}}
	admin := token(1, model.RoleAdmin)
	pengelola := token(seedUserBase, model.RolePengelola)

	// 1. Siapkan batch baru lalu buka war.
	var batch model.Batch
	c.must(http.MethodPost, fmt.Sprintf("/modules/%d/batches", moduleID), admin, map[string]any{
		"quota":     quota,
		"opens_at":  time.Now().Add(time.Minute).UTC(),
		"closes_at": time.Now().Add(time.Hour).UTC(),
	}, http.StatusCreated, &batch)
	c.must(http.MethodPatch, fmt.Sprintf("/batches/%d/status", batch.ID), admin, map[string]string{"status": "scheduled"}, http.StatusOK, nil)
	c.must(http.MethodPatch, fmt.Sprintf("/batches/%d/status", batch.ID), admin, map[string]string{"status": "open"}, http.StatusOK, nil)
	t.Logf("batch %d dibuka, kuota %d, %d peserta masing-masing menembak 2 kali serentak", batch.ID, quota, users)

	// 2. Semua peserta menembak dua kali di waktu yang sama.
	type result struct {
		userID int
		status int
		claim  model.Claim
		err    error
	}
	results := make(chan result, users*2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 1; i <= users; i++ {
		for attempt := 0; attempt < 2; attempt++ {
			wg.Add(1)
			go func(uid int) {
				defer wg.Done()
				<-start
				code, env, err := c.do(http.MethodPost, "/claims", token(uid, model.RolePeserta), map[string]int{"batch_id": batch.ID})
				r := result{userID: uid, status: code, err: err}
				if err == nil && code == http.StatusCreated {
					_ = json.Unmarshal(env.Data, &r.claim)
				}
				results <- r
			}(seedUserBase + i)
		}
	}
	began := time.Now()
	close(start)
	wg.Wait()
	close(results)
	elapsed := time.Since(began)

	counts := map[int]int{}
	created := map[int]int{}
	byStatus := map[string]int{}
	for r := range results {
		if r.err != nil {
			t.Fatalf("request user %d gagal: %v", r.userID, r.err)
		}
		counts[r.status]++
		if r.status == http.StatusCreated {
			created[r.userID]++
			byStatus[r.claim.Status]++
		}
	}
	t.Logf("%d request selesai dalam %s, status HTTP: %v, status klaim: %v", users*2, elapsed, counts, byStatus)

	// 3. Pastikan tidak ada overbooking dan tidak ada klaim ganda.
	if counts[http.StatusCreated] != users || counts[http.StatusConflict] != users {
		t.Fatalf("harus tepat %d sukses dan %d konflik, dapat %v", users, users, counts)
	}
	for uid, n := range created {
		if n != 1 {
			t.Fatalf("user %d mendapat %d klaim di batch yang sama", uid, n)
		}
	}
	wantPending := min(quota, users)
	if byStatus[model.ClaimPending] != wantPending || byStatus[model.ClaimWaitlisted] != users-wantPending {
		t.Fatalf("harus %d pending dan %d waitlisted, dapat %v", wantPending, users-wantPending, byStatus)
	}

	var after model.Batch
	c.must(http.MethodGet, fmt.Sprintf("/batches/%d", batch.ID), admin, nil, http.StatusOK, &after)
	if after.Allocated != wantPending || after.Remaining != quota-wantPending {
		t.Fatalf("allocated %d remaining %d, want %d dan %d", after.Allocated, after.Remaining, wantPending, quota-wantPending)
	}
	if got := len(c.claimsByStatus(batch.ID, model.ClaimPending, pengelola)); got != wantPending {
		t.Fatalf("di database ada %d klaim pending, want %d", got, wantPending)
	}

	// 4. Satu pemegang slot membatalkan, antrean terdepan harus naik ke pending.
	if users > quota {
		waitBefore := c.claimsByStatus(batch.ID, model.ClaimWaitlisted, pengelola)
		head := waitBefore[len(waitBefore)-1] // daftar urut terbaru dulu, antrean terdepan ada di akhir
		pending := c.claimsByStatus(batch.ID, model.ClaimPending, pengelola)
		victim := pending[0]

		c.must(http.MethodPost, fmt.Sprintf("/claims/%d/cancel", victim.ID), token(victim.UserID, model.RolePeserta), nil, http.StatusOK, nil)

		var promoted model.Claim
		c.must(http.MethodGet, fmt.Sprintf("/claims/%d", head.ID), pengelola, nil, http.StatusOK, &promoted)
		if promoted.Status != model.ClaimPending {
			t.Fatalf("antrean terdepan klaim %d berstatus %s, want pending", head.ID, promoted.Status)
		}
		c.must(http.MethodGet, fmt.Sprintf("/batches/%d", batch.ID), admin, nil, http.StatusOK, &after)
		if after.Allocated != quota {
			t.Fatalf("setelah promosi allocated %d, want %d", after.Allocated, quota)
		}
		t.Logf("klaim %d dibatalkan, klaim %d naik dari daftar tunggu, allocated tetap %d", victim.ID, head.ID, after.Allocated)
	}

	c.must(http.MethodPatch, fmt.Sprintf("/batches/%d/status", batch.ID), admin, map[string]string{"status": "closed"}, http.StatusOK, nil)
}
