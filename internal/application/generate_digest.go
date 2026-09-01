package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/yarburart/str3k0za-radar/internal/domain"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

type DigestService struct {
	userRepo *postgres.UserRepository
	graph    *domain.AttackGraph
	cweData  []domain.CWE

	// contiguous memory layout for TTPs to avoid pointer chasing
	ttpList  []domain.TTP
	aptToTTP map[string][]int // for lookup, instead graph
}

func NewDigestService(userRepo *postgres.UserRepository, graph *domain.AttackGraph, cweData []domain.CWE) *DigestService {
	d := &DigestService{
		userRepo: userRepo,
		graph:    graph,
		cweData:  cweData,
	}

	if graph != nil {
		d.ttpList = make([]domain.TTP, 0, len(graph.TTPs))
		aptToTTPMap := make(map[string][]int)

		// for deterministic order, less duplication
		ids := make([]string, 0, len(graph.TTPs))
		for id := range graph.TTPs {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		for _, id := range ids {
			ttpPtr := graph.TTPs[id]
			if ttpPtr == nil {
				continue
			}

			idx := len(d.ttpList)
			// dereference and copy to value slice for contiguous memory
			d.ttpList = append(d.ttpList, *ttpPtr)

			for _, aptID := range ttpPtr.RelatedAPTIDs {
				aptToTTPMap[aptID] = append(aptToTTPMap[aptID], idx)
			}
		}
		d.aptToTTP = aptToTTPMap
	}

	return d
}

func (d *DigestService) GenerateDigest(ctx context.Context, telegramID int64) (domain.Digest, error) {
	user, err := d.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return domain.Digest{}, fmt.Errorf("failed to fetch user for digest: %w", err)
	}

	cwe, err := d.getRandomCWE()
	if err != nil {
		return domain.Digest{}, err
	}

	ttp, err := d.getRandomTTP(user.Prefs.APTGroups)
	if err != nil {
		return domain.Digest{}, err
	}

	return domain.Digest{
		CWERand: cwe,
		TTPRand: *ttp,
	}, nil
}

// just digest into string, for ready tg message
func (d *DigestService) GenerateDigestMessage(ctx context.Context, telegramID int64) (string, error) {
	digest, err := d.GenerateDigest(ctx, telegramID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	// avoid some dynamic resizing and reallocations
	b.Grow(512)

	b.WriteString("Daily Threat Digest\n\n")
	b.WriteString(fmt.Sprintf("CWE-%d: %s\n", digest.CWERand.ID, digest.CWERand.Name))
	if digest.CWERand.Description != "" {
		b.WriteString(fmt.Sprintf("  %s\n", digest.CWERand.Description))
	}
	b.WriteString("\n")

	b.WriteString("TTP:\n")
	b.WriteString(fmt.Sprintf("%s: %s\n", digest.TTPRand.MitreID, digest.TTPRand.Name))
	if digest.TTPRand.Description != "" {
		b.WriteString(fmt.Sprintf("  %s\n", digest.TTPRand.Description))
	}
	b.WriteString("\n")

	b.WriteString("Related APT Groups:\n")
	if len(digest.TTPRand.RelatedAPTIDs) > 0 {
		for i, aptID := range digest.TTPRand.RelatedAPTIDs {
			if i > 0 {
				b.WriteString(", ")
			}
			if apt, ok := d.graph.APTs[aptID]; ok {
				b.WriteString(fmt.Sprintf("%s (%s)", apt.Name, apt.MitreID))
			} else {
				b.WriteString(aptID)
			}
		}
		b.WriteString("\n")
	} else {
		b.WriteString("None identified\n")
	}

	return b.String(), nil
}

func maybeTryRandIntN(n int) (int, error) {
	max := big.NewInt(int64(n))
	val, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("crypto rand failed: %w", err)
	}
	return int(val.Int64()), nil
}

func (d *DigestService) getRandomCWE() (domain.CWE, error) {
	if len(d.cweData) == 0 {
		return domain.CWE{}, errors.New("cannot select from an empty CWE slice")
	}
	idx, err := maybeTryRandIntN(len(d.cweData))
	if err != nil {
		return domain.CWE{}, err
	}

	return d.cweData[idx], nil
}

func (d *DigestService) getRandomTTP(aptFilter []string) (*domain.TTP, error) {
	if len(d.ttpList) == 0 {
		return nil, errors.New("cannot select from an empty TTP slice")
	}

	if len(aptFilter) == 0 {
		// any APT, so pick any random TTP
		idx, err := maybeTryRandIntN(len(d.ttpList))
		if err != nil {
			return nil, err
		}
		return &d.ttpList[idx], nil
	}

	// stack allocated slice to collect candidate indices, cuz sort.Ints take concrete []int
	candidates := make([]int, 0, 64)
	for _, aptID := range aptFilter {
		if indices, ok := d.aptToTTP[aptID]; ok {
			candidates = append(candidates, indices...)
		}
	}

	if len(candidates) == 0 {
		// if filter doesnt work, then as any APT
		idx, err := maybeTryRandIntN(len(d.ttpList))
		if err != nil {
			return nil, err
		}
		return &d.ttpList[idx], nil
	}

	// sort and deduplicate TTP, cuz many APTs use the same TTPs
	sort.Ints(candidates)
	n := 0
	for i := 0; i < len(candidates); i++ {
		if i == 0 || candidates[i] != candidates[i-1] {
			candidates[n] = candidates[i]
			n++
		}
	}
	candidates = candidates[:n]

	idx, err := maybeTryRandIntN(len(candidates))
	if err != nil {
		return nil, err
	}
	chosenIdx := candidates[idx]
	return &d.ttpList[chosenIdx], nil
}
