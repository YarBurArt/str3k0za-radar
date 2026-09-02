package mitre

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/yarburart/str3k0za-radar/internal/domain"
)

type threatGroupBundle struct {
	Values []threatGroupCard `json:"values"`
}

type threatGroupCard struct {
	Actor string `json:"actor"`
	Names []struct {
		Name string `json:"name"`
	} `json:"names"`
	Country []string `json:"country"`
}

type Loader struct {
	mitreFilePath    string
	thgCardsFilePath string
	aliasToCountry   map[string]string
}

func NewLoader(mitreFilePath, thgCardsFilePath string) *Loader {
	return &Loader{
		mitreFilePath:    mitreFilePath,
		thgCardsFilePath: thgCardsFilePath,
		aliasToCountry:   make(map[string]string, 1500),
	}
}

type stixExternalRef struct {
	SourceName string `json:"source_name"`
	ExternalID string `json:"external_id"`
	URL        string `json:"url"`
}

type intrusionSetObject struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Aliases      []string          `json:"aliases"`
	ExternalRefs []stixExternalRef `json:"external_references"`
}

type attackPatternObject struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	ExternalRefs []stixExternalRef `json:"external_references"`
}

type relObject struct {
	SourceRef        string `json:"source_ref"`
	TargetRef        string `json:"target_ref"`
	RelationshipType string `json:"relationship_type"`
}

// check docs here for mitre format: https://attack.mitre.org/resources/attack-data-and-tools/
// and here https://oasis-open.github.io/cti-documentation/stix/intro
// we are just loading needed data in to AttackGraph, since both entity have several connections to each other
func (l *Loader) Load() (*domain.AttackGraph, error) {
	if err := l.loadThreatGroupCards(); err != nil {
		log.Printf("Warning: %v", err)
	}

	file, err := os.Open(l.mitreFilePath) // to partial read
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// stream until objects key
	dec := json.NewDecoder(file)
	for {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if key, ok := t.(string); ok && key == "objects" {
			break
		}
	}

	if t, err := dec.Token(); err != nil || t != json.Delim('[') {
		return nil, fmt.Errorf("expected TTP [ after objects key")
	}

	graph := &domain.AttackGraph{
		APTs: make(map[string]*domain.APT, 200),
		TTPs: make(map[string]*domain.TTP, 1000),
	}

	stixToAPT := make(map[string]string, 200)
	stixToTTP := make(map[string]string, 1000)

	type rel struct {
		source, target string
	}
	var rels []rel

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}

		// to check type
		var probe struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}

		switch probe.Type {
		case "intrusion-set":
			var obj intrusionSetObject
			if err := json.Unmarshal(raw, &obj); err == nil {
				apt := l.parseAPT(obj)
				if apt.MitreID != "" {
					graph.APTs[apt.MitreID] = apt
					stixToAPT[obj.ID] = apt.MitreID
				}
			}
		case "attack-pattern":
			var obj attackPatternObject
			if err := json.Unmarshal(raw, &obj); err == nil {
				ttp := l.parseTTP(obj)
				if ttp.MitreID != "" {
					graph.TTPs[ttp.MitreID] = ttp
					stixToTTP[obj.ID] = ttp.MitreID
				}
			}
		case "relationship":
			var obj relObject // skip descriptions
			if err := json.Unmarshal(raw, &obj); err == nil {
				if obj.RelationshipType == "uses" {
					rels = append(rels, rel{obj.SourceRef, obj.TargetRef})
				}
			}
		}
	}
	for _, r := range rels {
		aptID, ok1 := stixToAPT[r.source]
		ttpID, ok2 := stixToTTP[r.target]
		if ok1 && ok2 {
			if apt := graph.APTs[aptID]; apt != nil {
				apt.TechniqueIDs = append(apt.TechniqueIDs, ttpID)
			}
			if ttp := graph.TTPs[ttpID]; ttp != nil {
				ttp.RelatedAPTIDs = append(ttp.RelatedAPTIDs, aptID)
			}
		}
	}

	return graph, nil
}

// check APT data here: https://apt.etda.or.th/cgi-bin/listgroups.cgi
func (l *Loader) loadThreatGroupCards() error {
	data, err := os.ReadFile(l.thgCardsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read tg cards: %w", err)
	}

	var bundle threatGroupBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return fmt.Errorf("failed to parse tg cards: %w", err)
	}

	for _, card := range bundle.Values {
		country := l.normalizeCountry(card.Country)
		if country == "UNKNOWN" {
			continue
		}

		if card.Actor != "" {
			l.aliasToCountry[strings.ToLower(strings.TrimSpace(card.Actor))] = country
		}
		for _, n := range card.Names {
			if n.Name != "" {
				l.aliasToCountry[strings.ToLower(strings.TrimSpace(n.Name))] = country
			}
		}
	}

	return nil
}

func (l *Loader) normalizeCountry(countries []string) string {
	if len(countries) == 0 {
		return "UNKNOWN"
	}

	c := strings.ToLower(strings.TrimSpace(countries[0]))
	if c == "[unknown]" || c == "unknown" {
		return "UNKNOWN"
	}

	mapping := map[string]string{
		"china": "CN", "russia": "RU", "iran": "IR", "north korea": "KP",
		"south korea": "KR", "united states": "US", "israel": "IL",
		"pakistan": "PK", "india": "IN", "vietnam": "VN", "turkey": "TR",
		"belgium": "BE", "colombia": "CO", "lebanon": "LB",
		"romania": "RO", "united kingdom": "GB",
	}

	if code, ok := mapping[c]; ok {
		return code
	}
	// so if not in map
	if len(c) >= 2 {
		return strings.ToUpper(c[:2])
	}

	return "UNKNOWN"
}

func (l *Loader) parseAPT(obj intrusionSetObject) *domain.APT {
	return &domain.APT{
		MitreID:       l.extractMitreID(obj.ExternalRefs),
		StixID:        obj.ID,
		Name:          obj.Name,
		SourceCountry: l.getCountry(obj.Name, obj.Aliases),
		AltNames:      obj.Aliases,
	}
}

func (l *Loader) parseTTP(obj attackPatternObject) *domain.TTP {
	refs := make([]string, 0, len(obj.ExternalRefs))
	for _, ref := range obj.ExternalRefs {
		if ref.URL != "" {
			refs = append(refs, ref.URL)
		}
	}

	return &domain.TTP{
		MitreID:     l.extractMitreID(obj.ExternalRefs),
		StixID:      obj.ID,
		Name:        obj.Name,
		Description: obj.Description,
		References:  refs,
	}
}

func (l *Loader) extractMitreID(refs []stixExternalRef) string {
	for i := range refs {
		if refs[i].SourceName == "mitre-attack" && refs[i].ExternalID != "" {
			return refs[i].ExternalID
		}
	}
	return ""
}

func (l *Loader) getCountry(name string, aliases []string) string {
	if c, ok := l.aliasToCountry[strings.ToLower(strings.TrimSpace(name))]; ok {
		return c
	}

	for _, alias := range aliases {
		if c, ok := l.aliasToCountry[strings.ToLower(strings.TrimSpace(alias))]; ok {
			return c
		}
	}

	return "UNKNOWN"
}
