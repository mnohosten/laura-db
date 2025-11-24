package changestream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/query"
	"github.com/mnohosten/laura-db/pkg/replication"
)

// OperationType represents the type of change operation
type OperationType string

const (
	OperationTypeInsert             OperationType = "insert"
	OperationTypeUpdate             OperationType = "update"
	OperationTypeDelete             OperationType = "delete"
	OperationTypeReplace            OperationType = "replace"
	OperationTypeInvalidate         OperationType = "invalidate"
	OperationTypeDropCollection     OperationType = "drop"
	OperationTypeDropDatabase       OperationType = "dropDatabase"
	OperationTypeRename             OperationType = "rename"
	OperationTypeCreateIndex        OperationType = "createIndex"
	OperationTypeDropIndex          OperationType = "dropIndex"
	OperationTypeCreateCollection   OperationType = "createCollection"
)

// ChangeEvent represents a single change in the database
type ChangeEvent struct {
	// ID is the resume token (oplog OpID) for this event
	ID ResumeToken `json:"_id"`

	// OperationType describes the type of operation
	OperationType OperationType `json:"operationType"`

	// Timestamp when the operation occurred
	Timestamp time.Time `json:"clusterTime"`

	// Namespace information
	Database   string `json:"db"`
	Collection string `json:"coll"`

	// DocumentKey contains the _id of the document
	DocumentKey map[string]interface{} `json:"documentKey,omitempty"`

	// FullDocument contains the full document for insert operations
	// or the current version after update (if fullDocument is set to "updateLookup")
	FullDocument map[string]interface{} `json:"fullDocument,omitempty"`

	// UpdateDescription contains information about updated fields
	UpdateDescription *UpdateDescription `json:"updateDescription,omitempty"`

	// For index operations
	IndexDefinition map[string]interface{} `json:"indexDefinition,omitempty"`
}

// UpdateDescription describes what was updated in an update operation
type UpdateDescription struct {
	UpdatedFields map[string]interface{} `json:"updatedFields"`
	RemovedFields []string               `json:"removedFields"`
}

// ResumeToken is an opaque token that can be used to resume a change stream
type ResumeToken struct {
	OpID replication.OpID `json:"opId"`
}

// FullDocumentOption controls when to include the full document in change events
type FullDocumentOption string

const (
	// FullDocumentDefault does not include full document (only for inserts)
	FullDocumentDefault FullDocumentOption = "default"

	// FullDocumentUpdateLookup includes full document after update operations
	FullDocumentUpdateLookup FullDocumentOption = "updateLookup"
)

// ChangeStreamOptions configures a change stream
type ChangeStreamOptions struct {
	// FullDocument controls when to return the full document
	FullDocument FullDocumentOption

	// ResumeAfter specifies a resume token to start after
	ResumeAfter *ResumeToken

	// StartAtOperationTime starts the stream at a specific timestamp
	StartAtOperationTime *time.Time

	// MaxAwaitTime is the maximum time to wait for new changes (default: 1 second)
	MaxAwaitTime time.Duration

	// BatchSize is the number of events to buffer (default: 100)
	BatchSize int

	// Pipeline is an aggregation pipeline to filter/transform events
	Pipeline []map[string]interface{}
}

// DefaultChangeStreamOptions returns default options
func DefaultChangeStreamOptions() *ChangeStreamOptions {
	return &ChangeStreamOptions{
		FullDocument: FullDocumentDefault,
		MaxAwaitTime: 1 * time.Second,
		BatchSize:    100,
	}
}

// ChangeStream represents an active change stream
type ChangeStream struct {
	oplog      *replication.Oplog
	database   string
	collection string
	options    *ChangeStreamOptions
	filter     *query.Query

	// Event channel
	events chan *ChangeEvent
	errors chan error

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Current position
	mu              sync.RWMutex
	currentResumeToken ResumeToken

	// State
	closed bool
}

// NewChangeStream creates a new change stream
func NewChangeStream(oplog *replication.Oplog, database, collection string, options *ChangeStreamOptions) *ChangeStream {
	if options == nil {
		options = DefaultChangeStreamOptions()
	}

	ctx, cancel := context.WithCancel(context.Background())

	cs := &ChangeStream{
		oplog:      oplog,
		database:   database,
		collection: collection,
		options:    options,
		events:     make(chan *ChangeEvent, options.BatchSize),
		errors:     make(chan error, 10),
		ctx:        ctx,
		cancel:     cancel,
		closed:     false,
	}

	// Initialize resume token
	if options.ResumeAfter != nil {
		cs.currentResumeToken = *options.ResumeAfter
	} else {
		// Start from current position
		cs.currentResumeToken = ResumeToken{OpID: oplog.GetCurrentID()}
	}

	return cs
}

// SetFilter sets a filter for change events
func (cs *ChangeStream) SetFilter(filter map[string]interface{}) error {
	if filter == nil {
		cs.filter = nil
		return nil
	}

	cs.filter = query.NewQuery(filter)
	return nil
}

// Start begins watching for changes
func (cs *ChangeStream) Start() error {
	cs.mu.Lock()
	if cs.closed {
		cs.mu.Unlock()
		return fmt.Errorf("change stream is closed")
	}
	cs.mu.Unlock()

	go cs.watchLoop()
	return nil
}

// watchLoop continuously polls the oplog for new entries
func (cs *ChangeStream) watchLoop() {
	ticker := time.NewTicker(cs.options.MaxAwaitTime)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			if err := cs.pollOplog(); err != nil {
				cs.errors <- err
			}
		}
	}
}

// pollOplog fetches new oplog entries and converts them to change events
func (cs *ChangeStream) pollOplog() error {
	cs.mu.RLock()
	currentToken := cs.currentResumeToken
	cs.mu.RUnlock()

	// Get new entries since last position
	entries, err := cs.oplog.GetEntriesSince(currentToken.OpID)
	if err != nil {
		return fmt.Errorf("failed to fetch oplog entries: %w", err)
	}

	// Convert entries to change events
	for _, entry := range entries {
		// Filter by database and collection
		if cs.database != "" && entry.Database != cs.database {
			continue
		}
		if cs.collection != "" && entry.Collection != cs.collection {
			continue
		}

		// Convert to change event
		event := cs.convertToChangeEvent(entry)
		if event == nil {
			continue // Skip unsupported operations
		}

		// Apply filter if set
		if cs.filter != nil && !cs.matchesFilter(event) {
			continue
		}

		// Apply pipeline transformations if set
		if len(cs.options.Pipeline) > 0 {
			event = cs.applyPipeline(event)
			if event == nil {
				continue // Filtered out by pipeline
			}
		}

		// Update resume token
		cs.mu.Lock()
		cs.currentResumeToken = event.ID
		cs.mu.Unlock()

		// Send event (non-blocking)
		select {
		case cs.events <- event:
		case <-cs.ctx.Done():
			return nil
		default:
			// Buffer full, drop oldest event (or could block)
			// For now, we'll try to send but not block indefinitely
			select {
			case cs.events <- event:
			case <-time.After(100 * time.Millisecond):
				// Skip this event if buffer is full
			}
		}
	}

	return nil
}

// convertToChangeEvent converts an oplog entry to a change event
func (cs *ChangeStream) convertToChangeEvent(entry *replication.OplogEntry) *ChangeEvent {
	event := &ChangeEvent{
		ID: ResumeToken{OpID: entry.OpID},
		Timestamp: entry.Timestamp,
		Database: entry.Database,
		Collection: entry.Collection,
	}

	// Map oplog operation type to change stream operation type
	switch entry.OpType {
	case replication.OpTypeInsert:
		event.OperationType = OperationTypeInsert
		event.FullDocument = entry.Document
		if docID, ok := entry.Document["_id"]; ok {
			event.DocumentKey = map[string]interface{}{"_id": docID}
		}

	case replication.OpTypeUpdate:
		event.OperationType = OperationTypeUpdate
		if entry.DocID != nil {
			event.DocumentKey = map[string]interface{}{"_id": entry.DocID}
		}
		// Parse update description
		event.UpdateDescription = cs.parseUpdateDescription(entry.Update)

	case replication.OpTypeDelete:
		event.OperationType = OperationTypeDelete
		if entry.DocID != nil {
			event.DocumentKey = map[string]interface{}{"_id": entry.DocID}
		}

	case replication.OpTypeCreateCollection:
		event.OperationType = OperationTypeCreateCollection

	case replication.OpTypeDropCollection:
		event.OperationType = OperationTypeDropCollection

	case replication.OpTypeCreateIndex:
		event.OperationType = OperationTypeCreateIndex
		event.IndexDefinition = entry.IndexDef

	case replication.OpTypeDropIndex:
		event.OperationType = OperationTypeDropIndex
		event.IndexDefinition = entry.IndexDef

	case replication.OpTypeNoop:
		// Skip noop operations
		return nil

	default:
		// Unknown operation type
		return nil
	}

	return event
}

// parseUpdateDescription extracts updated and removed fields from an update operation
func (cs *ChangeStream) parseUpdateDescription(update map[string]interface{}) *UpdateDescription {
	desc := &UpdateDescription{
		UpdatedFields: make(map[string]interface{}),
		RemovedFields: make([]string, 0),
	}

	// Handle $set operator
	if setOp, ok := update["$set"].(map[string]interface{}); ok {
		for k, v := range setOp {
			desc.UpdatedFields[k] = v
		}
	}

	// Handle $unset operator
	if unsetOp, ok := update["$unset"].(map[string]interface{}); ok {
		for k := range unsetOp {
			desc.RemovedFields = append(desc.RemovedFields, k)
		}
	}

	// Handle $inc operator
	if incOp, ok := update["$inc"].(map[string]interface{}); ok {
		for k, v := range incOp {
			desc.UpdatedFields[k] = v
		}
	}

	// Add other operators as needed

	return desc
}

// matchesFilter checks if a change event matches the filter
func (cs *ChangeStream) matchesFilter(event *ChangeEvent) bool {
	// Convert event to a document format for query evaluation
	docMap := map[string]interface{}{
		"operationType": string(event.OperationType),
		"database":      event.Database,
		"collection":    event.Collection,
	}

	if event.FullDocument != nil {
		docMap["fullDocument"] = event.FullDocument
	}

	if event.DocumentKey != nil {
		docMap["documentKey"] = event.DocumentKey
	}

	doc := document.NewDocumentFromMap(docMap)
	matches, err := cs.filter.Matches(doc)
	if err != nil {
		return false
	}
	return matches
}

// applyPipeline applies pipeline transformations to the event
func (cs *ChangeStream) applyPipeline(event *ChangeEvent) *ChangeEvent {
	// For now, we support basic $match stage
	// Full pipeline support would require aggregation integration
	for _, stage := range cs.options.Pipeline {
		if matchStage, ok := stage["$match"].(map[string]interface{}); ok {
			q := query.NewQuery(matchStage)

			// Convert event to document for matching
			docMap := map[string]interface{}{
				"operationType": string(event.OperationType),
				"database":      event.Database,
				"collection":    event.Collection,
			}
			doc := document.NewDocumentFromMap(docMap)

			matches, err := q.Matches(doc)
			if err != nil || !matches {
				return nil // Filtered out
			}
		}
	}

	return event
}

// Next returns the next change event (blocking)
func (cs *ChangeStream) Next(ctx context.Context) (*ChangeEvent, error) {
	select {
	case event := <-cs.events:
		return event, nil
	case err := <-cs.errors:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cs.ctx.Done():
		return nil, fmt.Errorf("change stream closed")
	}
}

// TryNext returns the next change event if available (non-blocking)
func (cs *ChangeStream) TryNext() (*ChangeEvent, error) {
	select {
	case event := <-cs.events:
		return event, nil
	case err := <-cs.errors:
		return nil, err
	default:
		return nil, nil // No event available
	}
}

// Events returns the channel of change events for direct consumption
func (cs *ChangeStream) Events() <-chan *ChangeEvent {
	return cs.events
}

// Errors returns the channel of errors
func (cs *ChangeStream) Errors() <-chan error {
	return cs.errors
}

// ResumeToken returns the current resume token
func (cs *ChangeStream) ResumeToken() ResumeToken {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.currentResumeToken
}

// Close closes the change stream
func (cs *ChangeStream) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.closed {
		return nil
	}

	cs.closed = true
	cs.cancel()
	close(cs.events)
	close(cs.errors)

	return nil
}
