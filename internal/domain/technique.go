package domain

type APT struct {
	MitreID       string
	StixID        string
	Name          string
	SourceCountry string
	AltNames      []string
	TechniqueIDs  []string
}

// TODO: also unregistred APT from like https://apt.etda.or.th/cgi-bin/listgroups.cgi , dont have enough TTP data yet

type TTP struct {
	MitreID       string
	StixID        string
	Name          string
	Description   string
	References    []string
	RelatedAPTIDs []string
}

// normalized data in memory for fast lookups
type AttackGraph struct {
	APTs map[string]*APT
	TTPs map[string]*TTP
}
