package news

import (
	"context"
	"sort"
	"strings"
	"time"
)

func recordStage(t *Tx, ctx context.Context, a Doc, from string, u Doc) error {
	to := str(a["status"])
	if from == to {
		return nil
	}
	return t.Audit(ctx, u, "article.transition", str(a["id"]), Doc{"fromStatus": from, "toStatus": to, "revision": a["revision"]})
}

func editorialStageMetrics(articles, audits []Doc, at time.Time) Doc {
	stages := []string{"new", "writing", "review", "rework", "approved", "published", "rejected"}
	events := map[string][]Doc{}
	legacy := map[string]string{"create": "new", "edit": "review", "submit": "review", "approve": "approved", "publish": "published", "unpublish": "review", "reject": "rejected", "rework": "rework"}
	for _, audit := range audits {
		target := str(audit["targetId"])
		detail := object(audit["detail"])
		if str(audit["action"]) == "article.transition" {
			events[target] = append(events[target], Doc{"at": audit["createdAt"], "from": detail["fromStatus"], "to": detail["toStatus"]})
			continue
		}
		if boolean(detail["stageRecorded"]) {
			continue
		}
		if status := legacy[strings.TrimPrefix(str(audit["action"]), "article.")]; status != "" {
			events[target] = append(events[target], Doc{"at": audit["createdAt"], "to": status})
		}
	}
	visits := map[string][]float64{}
	active := map[string][]float64{}
	reached := map[string]int{}
	transitions := map[string]int{}
	for _, a := range articles {
		ev := events[str(a["id"])]
		sort.SliceStable(ev, func(i, j int) bool { return parseTime(ev[i]["at"]).Before(parseTime(ev[j]["at"])) })
		stage := "new"
		entered := parseTime(a["createdAt"])
		if entered.IsZero() {
			entered = at
		}
		seen := map[string]bool{"new": true}
		for _, event := range ev {
			next := str(event["to"])
			t := parseTime(event["at"])
			if next == "" || t.IsZero() || t.Before(entered) || t.After(at) {
				continue
			}
			if next == stage {
				continue
			}
			visits[stage] = append(visits[stage], t.Sub(entered).Seconds())
			transitions[stage+"|"+next]++
			stage = next
			entered = t
			seen[stage] = true
		}
		current := str(a["status"])
		if current != "" && current != stage {
			stage = current
			entered = parseTime(a["updatedAt"])
			if entered.IsZero() || entered.After(at) {
				entered = at
			}
			seen[stage] = true
		}
		active[stage] = append(active[stage], at.Sub(entered).Seconds())
		for s := range seen {
			reached[s]++
		}
	}
	durations := []any{}
	funnel := []any{}
	for _, stage := range stages {
		values := visits[stage]
		sort.Float64s(values)
		total := 0.0
		for _, n := range values {
			total += n
		}
		average, p50, p90 := 0.0, 0.0, 0.0
		if len(values) > 0 {
			average = total / float64(len(values))
			p50 = values[(len(values)-1)/2]
			p90 = values[(len(values)-1)*9/10]
		}
		age := 0.0
		for _, n := range active[stage] {
			age += n
		}
		if len(active[stage]) > 0 {
			age /= float64(len(active[stage]))
		}
		durations = append(durations, Doc{"stage": stage, "completedVisits": len(values), "averageSeconds": average, "medianSeconds": p50, "p90Seconds": p90, "currentCount": len(active[stage]), "averageCurrentAgeSeconds": age})
		rate := 0.0
		if len(articles) > 0 {
			rate = float64(reached[stage]) / float64(len(articles))
		}
		funnel = append(funnel, Doc{"stage": stage, "count": reached[stage], "rate": rate})
	}
	keys := []string{}
	for key := range transitions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	flows := []any{}
	for _, key := range keys {
		parts := strings.Split(key, "|")
		flows = append(flows, Doc{"from": parts[0], "to": parts[1], "count": transitions[key]})
	}
	return Doc{"durations": durations, "funnel": funnel, "transitions": flows, "articleCount": len(articles), "measuredAt": at.UTC().Format(time.RFC3339Nano)}
}
