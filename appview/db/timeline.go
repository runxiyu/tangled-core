package db

import (
	"sort"
	"time"
)

type TimelineEvent struct {
	*Repo
	*Follow
	EventAt time.Time
}

func MakeTimeline(e Execer) ([]TimelineEvent, error) {
	var events []TimelineEvent

	repos, err := GetAllRepos(e)
	if err != nil {
		return nil, err
	}

	follows, err := GetAllFollows(e)
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		events = append(events, TimelineEvent{
			Repo:    &repo,
			Follow:  nil,
			EventAt: repo.Created,
		})
	}

	for _, follow := range follows {
		events = append(events, TimelineEvent{
			Repo:    nil,
			Follow:  &follow,
			EventAt: follow.FollowedAt,
		})
	}

	// Limit the slice to 100 events
	if len(events) > 50 {
		events = events[:50]
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].EventAt.After(events[j].EventAt)
	})

	return events, nil
}
