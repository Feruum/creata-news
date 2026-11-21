package achievements

import (
	"connectrpc.com/connect"
	"context"
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BadgeInterceptor hydrates every user-name projection in one query per RPC,
// including nested registration/attendance users. Applying it at the transport
// keeps public, admin, and organization responses consistent without N+1 reads.
func BadgeInterceptor(db *bun.DB) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			response, err := next(ctx, req)
			if err != nil {
				return nil, err
			}
			message, ok := response.Any().(proto.Message)
			if !ok {
				return response, nil
			}
			if err = HydrateBadges(ctx, db, message); err != nil {
				return nil, rpcError(err)
			}
			return response, nil
		}
	})
}

func HydrateBadges(ctx context.Context, db bun.IDB, message proto.Message) error {
	type target struct {
		message protoreflect.Message
		field   protoreflect.FieldDescriptor
		user    int32
	}
	var targets []target
	ids := map[int32]bool{}
	var walk func(protoreflect.Message)
	walk = func(m protoreflect.Message) {
		descriptor := m.Descriptor()
		field := descriptor.Fields().ByName("selected_badge")
		if field != nil && field.Message() != nil && field.Message().FullName() == "achievements.v1.NameBadge" {
			key := protoreflect.Name("user_id")
			if descriptor.FullName() == "users.v1.User" || descriptor.FullName() == "search.v1.SearchResult" {
				key = "id"
			}
			userField := descriptor.Fields().ByName(key)
			if descriptor.FullName() == "search.v1.SearchResult" && m.Get(descriptor.Fields().ByName("type")).Enum() != 3 {
				userField = nil
				m.Clear(field)
			}
			if userField != nil {
				user := int32(m.Get(userField).Int())
				if user > 0 {
					targets = append(targets, target{m, field, user})
					ids[user] = true
				}
			}
		}
		if field = descriptor.Fields().ByName("invited_by_selected_badge"); field != nil {
			userField := descriptor.Fields().ByName("invited_by_user_id")
			if userField != nil {
				user := int32(m.Get(userField).Int())
				if user > 0 {
					targets = append(targets, target{m, field, user})
					ids[user] = true
				}
			}
		}
		if field = descriptor.Fields().ByName("performed_by_selected_badge"); field != nil {
			userField := descriptor.Fields().ByName("performed_by")
			if userField != nil {
				user := int32(m.Get(userField).Int())
				if user > 0 {
					targets = append(targets, target{m, field, user})
					ids[user] = true
				}
			}
		}
		m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if f.IsMap() {
				if f.MapValue().Kind() == protoreflect.MessageKind {
					v.Map().Range(func(_ protoreflect.MapKey, value protoreflect.Value) bool { walk(value.Message()); return true })
				}
			} else if f.IsList() {
				if f.Kind() == protoreflect.MessageKind {
					list := v.List()
					for i := 0; i < list.Len(); i++ {
						walk(list.Get(i).Message())
					}
				}
			} else if f.Kind() == protoreflect.MessageKind {
				walk(v.Message())
			}
			return true
		})
	}
	walk(message.ProtoReflect())
	if len(targets) == 0 {
		return nil
	}
	users := make([]int32, 0, len(ids))
	for id := range ids {
		users = append(users, id)
	}
	var rows []struct {
		UserID        int32
		AchievementID int32
		Title         string
		IconURL       string
		AccentColor   string
	}
	err := db.NewRaw(`SELECT s.user_id,a.id AS achievement_id,a.title,a.badge_icon_url AS icon_url,a.accent_color
 FROM achievement_badge_selections s JOIN achievements a ON a.id=s.achievement_id
 JOIN user_achievements ua ON ua.user_id=s.user_id AND ua.achievement_id=s.achievement_id
 WHERE s.user_id IN (?) AND a.status IN (2,3) AND a.badge_icon_url<>'' AND NOT ua.suppressed AND (ua.manual OR ua.automatic)`, bun.In(users)).Scan(ctx, &rows)
	if err != nil {
		return err
	}
	badges := map[int32]*pb.NameBadge{}
	for _, r := range rows {
		badges[r.UserID] = &pb.NameBadge{AchievementId: r.AchievementID, Title: r.Title, IconUrl: r.IconURL, AccentColor: r.AccentColor}
	}
	for _, target := range targets {
		if badge := badges[target.user]; badge != nil {
			target.message.Set(target.field, protoreflect.ValueOfMessage(badge.ProtoReflect()))
		} else {
			target.message.Clear(target.field)
		}
	}
	return nil
}
