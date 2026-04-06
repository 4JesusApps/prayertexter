package service

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

func (s *Service) memberDelete(ctx context.Context, mem *model.Member) error {
	if err := s.deleteMember(ctx, mem.Phone); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := s.removeIntercessor(ctx, mem); err != nil {
			return err
		}
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgRemoveUser)
}

func (s *Service) removeIntercessor(ctx context.Context, mem *model.Member) error {
	phones, err := s.getIntercessorPhones(ctx)
	if err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err := s.putIntercessorPhones(ctx, phones); err != nil {
		return err
	}

	return s.moveActivePrayer(ctx, mem)
}

func (s *Service) moveActivePrayer(ctx context.Context, mem *model.Member) error {
	isActive, err := s.isPrayerActive(ctx, mem.Phone)
	if err != nil {
		return err
	}

	if isActive {
		pryr, err := s.getActivePrayer(ctx, mem.Phone)
		if err != nil {
			return err
		}

		if err = s.deleteActivePrayer(ctx, mem.Phone); err != nil {
			return err
		}

		var id string
		id, err = model.GenerateID()
		if err != nil {
			return err
		}
		pryr.IntercessorPhone = id
		pryr.Intercessor = model.Member{}

		return s.putQueuedPrayer(ctx, pryr)
	}

	return nil
}
