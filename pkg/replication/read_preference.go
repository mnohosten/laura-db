package replication

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ReadPreferenceMode determines where reads are routed
type ReadPreferenceMode int

const (
	// ReadPrimary - all reads from primary only (default)
	ReadPrimary ReadPreferenceMode = iota

	// ReadPrimaryPreferred - read from primary if available, else secondary
	ReadPrimaryPreferred

	// ReadSecondary - read from secondary only (error if no secondaries)
	ReadSecondary

	// ReadSecondaryPreferred - read from secondary if available, else primary
	ReadSecondaryPreferred

	// ReadNearest - read from node with lowest latency
	ReadNearest
)

// String returns the string representation of the read preference mode
func (m ReadPreferenceMode) String() string {
	switch m {
	case ReadPrimary:
		return "primary"
	case ReadPrimaryPreferred:
		return "primaryPreferred"
	case ReadSecondary:
		return "secondary"
	case ReadSecondaryPreferred:
		return "secondaryPreferred"
	case ReadNearest:
		return "nearest"
	default:
		return "unknown"
	}
}

// ReadPreference defines how reads should be routed in a replica set
type ReadPreference struct {
	Mode ReadPreferenceMode

	// MaxStalenessSeconds - max acceptable lag for secondary reads (0 = no limit)
	MaxStalenessSeconds int

	// Tags - optional tag filters for selecting specific nodes
	Tags map[string]string

	mu sync.RWMutex
}

// NewReadPreference creates a new read preference with the specified mode
func NewReadPreference(mode ReadPreferenceMode) *ReadPreference {
	return &ReadPreference{
		Mode:                mode,
		MaxStalenessSeconds: 0,
		Tags:                make(map[string]string),
	}
}

// Primary returns a read preference for primary reads only
func Primary() *ReadPreference {
	return NewReadPreference(ReadPrimary)
}

// PrimaryPreferred returns a read preference for primary-preferred reads
func PrimaryPreferred() *ReadPreference {
	return NewReadPreference(ReadPrimaryPreferred)
}

// Secondary returns a read preference for secondary reads only
func Secondary() *ReadPreference {
	return NewReadPreference(ReadSecondary)
}

// SecondaryPreferred returns a read preference for secondary-preferred reads
func SecondaryPreferred() *ReadPreference {
	return NewReadPreference(ReadSecondaryPreferred)
}

// Nearest returns a read preference for nearest node reads
func Nearest() *ReadPreference {
	return NewReadPreference(ReadNearest)
}

// WithMaxStaleness sets the maximum staleness for secondary reads
func (rp *ReadPreference) WithMaxStaleness(seconds int) *ReadPreference {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.MaxStalenessSeconds = seconds
	return rp
}

// WithTags sets tag filters for node selection
func (rp *ReadPreference) WithTags(tags map[string]string) *ReadPreference {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.Tags = tags
	return rp
}

// GetMode returns the read preference mode
func (rp *ReadPreference) GetMode() ReadPreferenceMode {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.Mode
}

// GetMaxStaleness returns the maximum staleness in seconds
func (rp *ReadPreference) GetMaxStaleness() int {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.MaxStalenessSeconds
}

// GetTags returns the tag filters
func (rp *ReadPreference) GetTags() map[string]string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Return a copy to avoid external mutations
	tags := make(map[string]string, len(rp.Tags))
	for k, v := range rp.Tags {
		tags[k] = v
	}
	return tags
}

// String returns a string representation of the read preference
func (rp *ReadPreference) String() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	s := fmt.Sprintf("ReadPreference{mode=%s", rp.Mode)
	if rp.MaxStalenessSeconds > 0 {
		s += fmt.Sprintf(", maxStaleness=%ds", rp.MaxStalenessSeconds)
	}
	if len(rp.Tags) > 0 {
		s += fmt.Sprintf(", tags=%v", rp.Tags)
	}
	s += "}"
	return s
}

// NodeCandidate represents a candidate node for read operations
type NodeCandidate struct {
	NodeID   string
	Role     NodeRole
	State    NodeState
	Lag      time.Duration
	Latency  time.Duration
	Tags     map[string]string
}

// ReadPreferenceSelector selects appropriate nodes based on read preference
type ReadPreferenceSelector struct {
	replicaSet *ReplicaSet
	mu         sync.RWMutex
}

// NewReadPreferenceSelector creates a new read preference selector
func NewReadPreferenceSelector(rs *ReplicaSet) *ReadPreferenceSelector {
	return &ReadPreferenceSelector{
		replicaSet: rs,
	}
}

// SelectNode selects a node based on the read preference
func (s *ReadPreferenceSelector) SelectNode(ctx context.Context, pref *ReadPreference) (string, error) {
	if pref == nil {
		pref = Primary() // Default to primary
	}

	candidates := s.getCandidates()
	if len(candidates) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	// Filter candidates based on read preference mode
	switch pref.GetMode() {
	case ReadPrimary:
		return s.selectPrimary(candidates)

	case ReadPrimaryPreferred:
		node, err := s.selectPrimary(candidates)
		if err == nil {
			return node, nil
		}
		// Fall back to secondary
		return s.selectSecondary(candidates, pref)

	case ReadSecondary:
		return s.selectSecondary(candidates, pref)

	case ReadSecondaryPreferred:
		node, err := s.selectSecondary(candidates, pref)
		if err == nil {
			return node, nil
		}
		// Fall back to primary
		return s.selectPrimary(candidates)

	case ReadNearest:
		return s.selectNearest(candidates, pref)

	default:
		return "", fmt.Errorf("unknown read preference mode: %v", pref.GetMode())
	}
}

// getCandidates returns all available node candidates
func (s *ReadPreferenceSelector) getCandidates() []*NodeCandidate {
	members := s.replicaSet.GetMembers()
	candidates := make([]*NodeCandidate, 0, len(members))

	for _, member := range members {
		// Only include healthy nodes
		if member.State != StateHealthy {
			continue
		}

		candidates = append(candidates, &NodeCandidate{
			NodeID:  member.NodeID,
			Role:    member.Role,
			State:   member.State,
			Lag:     member.Lag,
			Latency: 0, // Would be measured in real implementation
			Tags:    make(map[string]string), // Would come from node config
		})
	}

	return candidates
}

// selectPrimary selects the primary node
func (s *ReadPreferenceSelector) selectPrimary(candidates []*NodeCandidate) (string, error) {
	for _, candidate := range candidates {
		if candidate.Role == RolePrimary {
			return candidate.NodeID, nil
		}
	}
	return "", fmt.Errorf("no primary node available")
}

// selectSecondary selects a secondary node
func (s *ReadPreferenceSelector) selectSecondary(candidates []*NodeCandidate, pref *ReadPreference) (string, error) {
	// Filter for secondary nodes
	secondaries := make([]*NodeCandidate, 0)
	for _, candidate := range candidates {
		if candidate.Role != RoleSecondary {
			continue
		}

		// Check max staleness
		maxStaleness := pref.GetMaxStaleness()
		if maxStaleness > 0 {
			if candidate.Lag > time.Duration(maxStaleness)*time.Second {
				continue // Too stale
			}
		}

		// Check tags (simplified - in real impl would do more complex matching)
		tags := pref.GetTags()
		if len(tags) > 0 {
			matches := true
			for k, v := range tags {
				if candidate.Tags[k] != v {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		secondaries = append(secondaries, candidate)
	}

	if len(secondaries) == 0 {
		return "", fmt.Errorf("no suitable secondary nodes available")
	}

	// Randomly select one of the secondaries for load balancing
	return secondaries[rand.Intn(len(secondaries))].NodeID, nil
}

// selectNearest selects the nearest node (lowest latency)
func (s *ReadPreferenceSelector) selectNearest(candidates []*NodeCandidate, pref *ReadPreference) (string, error) {
	// Filter candidates based on max staleness and tags
	eligible := make([]*NodeCandidate, 0)

	for _, candidate := range candidates {
		// Check max staleness for non-primary nodes
		maxStaleness := pref.GetMaxStaleness()
		if candidate.Role != RolePrimary && maxStaleness > 0 {
			if candidate.Lag > time.Duration(maxStaleness)*time.Second {
				continue
			}
		}

		// Check tags
		tags := pref.GetTags()
		if len(tags) > 0 {
			matches := true
			for k, v := range tags {
				if candidate.Tags[k] != v {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		eligible = append(eligible, candidate)
	}

	if len(eligible) == 0 {
		return "", fmt.Errorf("no suitable nodes available")
	}

	// Find node with lowest latency
	// In this simplified implementation, we'll just pick randomly
	// In a real implementation, would measure actual latency
	return eligible[rand.Intn(len(eligible))].NodeID, nil
}

// ReadRouter wraps a replica set and routes reads based on read preference
type ReadRouter struct {
	replicaSet *ReplicaSet
	selector   *ReadPreferenceSelector
	mu         sync.RWMutex
}

// NewReadRouter creates a new read router
func NewReadRouter(rs *ReplicaSet) *ReadRouter {
	return &ReadRouter{
		replicaSet: rs,
		selector:   NewReadPreferenceSelector(rs),
	}
}

// ReadDocument reads a document with the specified read preference
func (r *ReadRouter) ReadDocument(ctx context.Context, collName string, filter map[string]interface{}, pref *ReadPreference) (map[string]interface{}, error) {
	// Select appropriate node
	_, err := r.selector.SelectNode(ctx, pref)
	if err != nil {
		return nil, fmt.Errorf("failed to select node: %w", err)
	}

	// Route to appropriate node
	// In a real implementation, would route to the actual node
	// For now, we'll read from the local database
	coll := r.replicaSet.db.Collection(collName)
	doc, err := coll.FindOne(filter)
	if err != nil {
		return nil, err
	}

	return doc.ToMap(), nil
}

// ReadDocuments reads multiple documents with the specified read preference
func (r *ReadRouter) ReadDocuments(ctx context.Context, collName string, filter map[string]interface{}, pref *ReadPreference) ([]map[string]interface{}, error) {
	// Select appropriate node
	_, err := r.selector.SelectNode(ctx, pref)
	if err != nil {
		return nil, fmt.Errorf("failed to select node: %w", err)
	}

	// Route to appropriate node
	// In a real implementation, would route to the actual node
	// For now, we'll read from the local database
	coll := r.replicaSet.db.Collection(collName)
	docs, err := coll.Find(filter)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		result[i] = doc.ToMap()
	}

	return result, nil
}

// GetSelectedNode returns the node that would be selected for the given read preference
func (r *ReadRouter) GetSelectedNode(ctx context.Context, pref *ReadPreference) (string, error) {
	return r.selector.SelectNode(ctx, pref)
}
