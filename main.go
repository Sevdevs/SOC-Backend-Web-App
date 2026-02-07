package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

type Incident struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	Owner     string    `json:"owner"`
	Tags      []string  `json:"tags"`
	IOCs      []string  `json:"iocs"`
	Notes     []Note    `json:"notes"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type IncidentInput struct {
	Title    string   `json:"title"`
	Severity string   `json:"severity"`
	Status   string   `json:"status"`
	Owner    string   `json:"owner"`
	Tags     []string `json:"tags"`
	IOCs     []string `json:"iocs"`
}

type IncidentUpdate struct {
	Severity string `json:"severity"`
	Status   string `json:"status"`
	Owner    string `json:"owner"`
}

type NoteInput struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

type IncidentStore struct {
	mu        sync.RWMutex
	incidents map[string]*Incident
	order     []string
	counter   int
}

func newIncidentStore() *IncidentStore {
	store := &IncidentStore{
		incidents: make(map[string]*Incident),
		order:     []string{},
		counter:   1000,
	}

	seed := []IncidentInput{
		{
			Title:    "Suspicious OAuth consent grant",
			Severity: "High",
			Status:   "Investigating",
			Owner:    "SOC Tier 2",
			Tags:     []string{"identity", "cloud"},
			IOCs:     []string{"a1f4b9f", "login.live.com"},
		},
		{
			Title:    "Unusual lateral movement across finance segment",
			Severity: "Critical",
			Status:   "Contained",
			Owner:    "IR Lead",
			Tags:     []string{"lateral", "endpoint"},
			IOCs:     []string{"10.22.18.9", "svc_backup"},
		},
		{
			Title:    "Phishing campaign targeting HR",
			Severity: "Medium",
			Status:   "New",
			Owner:    "SOC Tier 1",
			Tags:     []string{"phishing", "email"},
			IOCs:     []string{"payroll-update.com"},
		},
	}

	for _, incident := range seed {
		store.create(incident)
	}

	return store
}

func (s *IncidentStore) list() []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Incident, 0, len(s.order))
	for _, id := range s.order {
		incident := s.incidents[id]
		if incident == nil {
			continue
		}
		items = append(items, *incident)
	}
	return items
}

func filterIncidents(items []Incident, severity, status, query string) []Incident {
	severity = strings.TrimSpace(strings.ToLower(severity))
	status = strings.TrimSpace(strings.ToLower(status))
	query = strings.TrimSpace(strings.ToLower(query))

	if severity == "" && status == "" && query == "" {
		return items
	}

	filtered := make([]Incident, 0, len(items))
	for _, incident := range items {
		if severity != "" && strings.ToLower(incident.Severity) != severity {
			continue
		}
		if status != "" && strings.ToLower(incident.Status) != status {
			continue
		}
		if query != "" && !matchesQuery(incident, query) {
			continue
		}
		filtered = append(filtered, incident)
	}

	return filtered
}

func matchesQuery(incident Incident, query string) bool {
	if strings.Contains(strings.ToLower(incident.Title), query) {
		return true
	}
	if strings.Contains(strings.ToLower(incident.Owner), query) {
		return true
	}
	for _, tag := range incident.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	for _, ioc := range incident.IOCs {
		if strings.Contains(strings.ToLower(ioc), query) {
			return true
		}
	}
	return false
}

func (s *IncidentStore) get(id string) (*Incident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[id]
	if !ok {
		return nil, false
	}
	copyIncident := *incident
	return &copyIncident, true
}

func (s *IncidentStore) create(input IncidentInput) Incident {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	id := "INC-" + padInt(s.counter)
	newIncident := &Incident{
		ID:        id,
		Title:     input.Title,
		Severity:  fallback(input.Severity, "Medium"),
		Status:    fallback(input.Status, "New"),
		Owner:     fallback(input.Owner, "Unassigned"),
		Tags:      sanitizeSlice(input.Tags),
		IOCs:      sanitizeSlice(input.IOCs),
		Notes:     []Note{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	s.incidents[id] = newIncident
	s.order = append([]string{id}, s.order...)

	return *newIncident
}

func (s *IncidentStore) update(id string, input IncidentUpdate) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	incident, ok := s.incidents[id]
	if !ok {
		return Incident{}, errors.New("incident not found")
	}

	if input.Severity != "" {
		incident.Severity = input.Severity
	}
	if input.Status != "" {
		incident.Status = input.Status
	}
	if input.Owner != "" {
		incident.Owner = input.Owner
	}
	incident.UpdatedAt = time.Now().UTC()

	return *incident, nil
}

func (s *IncidentStore) addNote(id string, input NoteInput) (Incident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	incident, ok := s.incidents[id]
	if !ok {
		return Incident{}, errors.New("incident not found")
	}
	if strings.TrimSpace(input.Body) == "" {
		return Incident{}, errors.New("note body required")
	}

	note := Note{
		ID:        "NOTE-" + padInt(len(incident.Notes)+1),
		Body:      input.Body,
		Author:    fallback(input.Author, "Analyst"),
		CreatedAt: time.Now().UTC(),
	}
	incident.Notes = append([]Note{note}, incident.Notes...)
	incident.UpdatedAt = time.Now().UTC()

	return *incident, nil
}

func padInt(value int) string {
	if value < 10 {
		return "000" + itoa(value)
	}
	if value < 100 {
		return "00" + itoa(value)
	}
	if value < 1000 {
		return "0" + itoa(value)
	}
	return itoa(value)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func sanitizeSlice(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return clean
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func readJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := newIncidentStore()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/incidents", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			severity := r.URL.Query().Get("severity")
			status := r.URL.Query().Get("status")
			query := r.URL.Query().Get("q")
			items := filterIncidents(store.list(), severity, status, query)
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		case http.MethodPost:
			var input IncidentInput
			if err := readJSON(r, &input); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			if strings.TrimSpace(input.Title) == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
				return
			}
			incident := store.create(input)
			writeJSON(w, http.StatusCreated, incident)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/incidents/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/incidents/")
		parts := strings.Split(path, "/")
		id := parts[0]
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				incident, ok := store.get(id)
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeJSON(w, http.StatusOK, incident)
			case http.MethodPut:
				var input IncidentUpdate
				if err := readJSON(r, &input); err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
					return
				}
				incident, err := store.update(id, input)
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeJSON(w, http.StatusOK, incident)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		if len(parts) == 2 && parts[1] == "notes" {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var input NoteInput
			if err := readJSON(r, &input); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			incident, err := store.addNote(id, input)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, incident)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	mux.Handle("/", http.FileServer(http.Dir("./static")))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("listening on http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
