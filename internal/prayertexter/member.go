package prayertexter

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

func memberDelete(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	if err := mem.Delete(ctx, ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := removeIntercessor(ctx, ddbClnt, mem); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgRemoveUser)
}

func removeIntercessor(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err := phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	// Moves an active prayer from the intercessor being removed to the Prayer queue. This is done to ensure that the
	// prayer eventually gets assigned to another intercessor.
	return moveActivePrayer(ctx, ddbClnt, mem)
}

func moveActivePrayer(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	isActive, err := object.IsPrayerActive(ctx, ddbClnt, mem.Phone)
	if err != nil {
		return err
	}

	if isActive {
		pryr := object.Prayer{IntercessorPhone: mem.Phone}
		if err = pryr.Get(ctx, ddbClnt, false); err != nil {
			return err
		}

		if err = pryr.Delete(ctx, ddbClnt, false); err != nil {
			return err
		}

		// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in
		// place if the intercessors phone number until there is an available intercessor, at which time the ID will get
		// changed to the available intercessors phone number.
		var id string
		id, err = utility.GenerateID()
		if err != nil {
			return err
		}
		pryr.IntercessorPhone = id
		pryr.Intercessor = object.Member{}

		return pryr.Put(ctx, ddbClnt, true)
	}

	return nil
}
