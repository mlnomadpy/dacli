package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Queue is an ordered checklist with a cursor. dacli never executes a step;
// the agent runs it and advances. The owner is the only writer — the cursor
// is mutable state, and an unowned cursor was the C2 defect.
type Queue struct {
	Slug   string
	Title  string
	Owner  string
	Cursor int
	Halted string // non-empty = halted, with the reason
	Steps  []string
	Doc    *mdstore.Doc
	Path   string
}

var stepRe = regexp.MustCompile(`^\s*\d+\.\s+(.*)$`)

// CreateQueue writes queues/<slug>.md owned by the creator.
func CreateQueue(w *workspace.Workspace, actor, slug, title string, steps []string) (*Queue, error) {
	if slug == "" {
		slug = Slugify(title)
	}
	path := w.QueuePath(slug)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("queue %q already exists", slug)
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("a queue with no steps is a title, not a checklist")
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", "q-"+slug)
	d.Front.Set("kind", string(model.KindQueue))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("owner", actor)
	d.Front.Set("cursor", "0")

	var body strings.Builder
	for i, s := range steps {
		fmt.Fprintf(&body, "%d. %s\n", i+1, s)
	}
	if title == "" {
		title = slug
	}
	d.Sections = []mdstore.Section{
		{Level: 1, Title: title, Content: ""},
		{Level: 2, Title: "Steps", Content: body.String()},
	}
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	return LoadQueue(w, slug)
}

// LoadQueue reads one queue.
func LoadQueue(w *workspace.Workspace, slug string) (*Queue, error) {
	path := w.QueuePath(slug)
	d, err := mdstore.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Ref: "queue/" + slug}
		}
		return nil, err
	}
	q := &Queue{Slug: slug, Doc: d, Path: path}
	q.Owner, _ = d.Front.Get("owner")
	q.Halted, _ = d.Front.Get("halted")
	if c, ok := d.Front.Get("cursor"); ok {
		q.Cursor, _ = strconv.Atoi(c)
	}
	for _, s := range d.Sections {
		if s.Level == 1 {
			q.Title = s.Title
		}
		if strings.EqualFold(s.Title, "Steps") {
			for _, line := range strings.Split(s.Content, "\n") {
				if m := stepRe.FindStringSubmatch(line); m != nil {
					q.Steps = append(q.Steps, m[1])
				}
			}
		}
	}
	return q, nil
}

// ListQueues returns every queue.
func ListQueues(w *workspace.Workspace) ([]*Queue, error) {
	entries, err := os.ReadDir(w.QueuesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no queues dir yet is not an error
		}
		return nil, err // a real I/O/permission failure must not read as "empty"
	}
	var out []*Queue
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if q, err := LoadQueue(w, strings.TrimSuffix(e.Name(), ".md")); err == nil {
			out = append(out, q)
		}
	}
	return out, nil
}

// Next returns the current step, or done=true when the queue is complete.
func (q *Queue) Next() (step string, done bool) {
	if q.Cursor >= len(q.Steps) {
		return "", true
	}
	return q.Steps[q.Cursor], false
}

// Advance moves the cursor, or halts the queue with a reason. Ownership is
// checked by the caller (identity lives above this layer).
func Advance(q *Queue, failReason string) error {
	if q.Halted != "" {
		return fmt.Errorf("queue is halted: %s (edit %s to resume)", q.Halted, filepath.Base(q.Path))
	}
	if failReason != "" {
		q.Doc.Front.Set("halted", failReason)
	} else {
		if q.Cursor >= len(q.Steps) {
			return fmt.Errorf("queue already complete")
		}
		q.Cursor++
		q.Doc.Front.Set("cursor", strconv.Itoa(q.Cursor))
	}
	return mdstore.WriteFile(q.Path, q.Doc)
}
