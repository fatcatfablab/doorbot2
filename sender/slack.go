package sender

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/slack-go/slack"
)

const (
	badgeEarnedFmt = ":tada: Achievement unlocked! You get the %s medal: %s"
	slackInitMsg   = `Primordial abyss abandoned. Initiating connection to Slack. ` +
		`Resuming sentinel duty. New arrivals shall be announced once more.`
)

var (
	totalsConf = map[uint]badge{
		0:   {badge: ":fatcat:", msg: ""},
		5:   {badge: ":fatcat-yellow:", msg: "UNO"},
		29:  {badge: ":fatcat-green:", msg: "TEENSY"},
		98:  {badge: ":fatcat-blue:", msg: "RPI"},
		363: {badge: ":fatcat-pink:", msg: "COMMUNITY"},
		498: {badge: ":fatcat-red:", msg: "CORE"},
		998: {badge: ":fatcat-black:", msg: "PILLAR"},
	}

	streaksConf = map[uint]badge{
		0:   {badge: ":cat2:", msg: ""},
		4:   {badge: ":black_cat:", msg: "One dedicated cat!"},
		13:  {badge: ":rat:", msg: "Lab cat to lab rat!"},
		30:  {badge: ":tiger2:", msg: "What the ...?"},
		182: {badge: ":leopard:", msg: "Do you sleep here?"},
		365: {badge: ":house_with_garden:", msg: "You DO live here! Welcome home."},
	}
)

type badge struct {
	badge string
	msg   string
}

type SlackSender struct {
	client  *slack.Client
	channel string
}

func NewSlack(channel, token string) *SlackSender {
	client := slack.New(token)
	c, ts, err := client.PostMessage(
		channel,
		slack.MsgOptionText(slackInitMsg, false),
	)
	if err != nil {
		log.Printf("error posting to slack: %s", err)
	} else {
		log.Printf("slack message posted to %s at %s", c, ts)
	}

	return &SlackSender{client: client, channel: channel}
}

func (s *SlackSender) Post(ctx context.Context, stats db.Stats) error {
	c, ts, err := s.client.PostMessageContext(
		ctx,
		s.channel,
		slack.MsgOptionText(statsToString(stats), false),
	)
	if err != nil {
		return fmt.Errorf("error posting msg to slack: %w", err)
	}

	log.Printf("Msg posted to %s (%s) at %s", s.channel, c, ts)
	return nil
}

func statsToString(stats db.Stats) string {
	tBadge, tEarned := getTotalBadge(stats.Total)
	sBadge, sEarned := getStreakBadge(stats.Streak)

	var sb strings.Builder
	fmt.Fprintf(
		&sb,
		"%s %s %d %s %d",
		stats.Name,
		tBadge.badge,
		stats.Total,
		sBadge.badge,
		stats.Streak,
	)

	if tEarned && stats.Total > 1 {
		fmt.Fprintf(&sb, "\n"+badgeEarnedFmt, tBadge.msg, tBadge.badge)
	}

	if sEarned && stats.Streak > 1 {
		fmt.Fprintf(&sb, "\n%s", sBadge.msg)
	}

	return sb.String()
}

func getTotalBadge(total uint) (badge, bool) {
	return findBadge(total, totalsConf)
}

func getStreakBadge(streak uint) (badge, bool) {
	return findBadge(streak, streaksConf)
}

func findBadge(num uint, conf map[uint]badge) (badge, bool) {
	var b badge
	var earned bool

	thresholds := slices.Collect(maps.Keys(conf))
	slices.Sort(thresholds)
	slices.Reverse(thresholds)
	for _, t := range thresholds {
		if num > t {
			b = conf[uint(t)]
			if num == t+1 {
				earned = true
			}
			break
		}
	}
	return b, earned
}
