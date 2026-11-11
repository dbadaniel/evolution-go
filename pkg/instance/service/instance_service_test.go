package instance_service

import (
	"context"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func TestClassifyAccountTypeUsesStoredBusinessNameWithoutLookup(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)
	lookupCalled := false

	result := classifyAccountType(context.Background(), jid, "  Loja Expertsa  ", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		lookupCalled = true
		return nil, errors.New("lookup must not be called")
	})

	if lookupCalled {
		t.Fatal("business profile lookup was called even though a stored business name was available")
	}
	assertAccountType(t, result, AccountTypeBusiness, true)
	if result.BusinessName != "Loja Expertsa" {
		t.Fatalf("BusinessName = %q, want %q", result.BusinessName, "Loja Expertsa")
	}
	if result.JID != jid.String() {
		t.Fatalf("JID = %q, want %q", result.JID, jid.String())
	}
}

func TestClassifyAccountTypeUsesBusinessProfile(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(_ context.Context, gotJID types.JID) (*types.BusinessProfile, error) {
		if gotJID != jid {
			t.Fatalf("lookup JID = %s, want %s", gotJID, jid)
		}
		return &types.BusinessProfile{JID: jid}, nil
	})

	assertAccountType(t, result, AccountTypeBusiness, true)
}

func TestClassifyAccountTypeRejectsMismatchedBusinessProfile(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)
	otherJID := types.NewJID("5511888888888", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		return &types.BusinessProfile{JID: otherJID}, nil
	})

	assertUnknownAccountType(t, result)
}

func TestClassifyAccountTypeTreatsMissingBusinessProfileAsPersonal(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		return nil, &whatsmeow.ElementMissingError{Tag: "business_profile", In: "response to business profile query"}
	})

	assertAccountType(t, result, AccountTypePersonal, false)
}

func TestClassifyAccountTypeTreatsNotFoundBusinessProfileAsPersonal(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		return nil, whatsmeow.ErrIQNotFound
	})

	assertAccountType(t, result, AccountTypePersonal, false)
}

func TestClassifyAccountTypeKeepsTransientFailureUnknown(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		return nil, context.DeadlineExceeded
	})

	assertUnknownAccountType(t, result)
}

func TestClassifyAccountTypeKeepsNilProfileUnknown(t *testing.T) {
	jid := types.NewJID("5511999999999", types.DefaultUserServer)

	result := classifyAccountType(context.Background(), jid, "", func(context.Context, types.JID) (*types.BusinessProfile, error) {
		return nil, nil
	})

	assertUnknownAccountType(t, result)
}

func TestAccountTypeRejectsNilInstance(t *testing.T) {
	service := instances{clientPointer: map[string]*whatsmeow.Client{}}

	_, err := service.AccountType(context.Background(), nil)
	if err == nil || err.Error() != "instance not found" {
		t.Fatalf("AccountType() error = %v, want instance not found", err)
	}
}

func assertAccountType(t *testing.T, result *AccountTypeStruct, wantType string, wantBusiness bool) {
	t.Helper()
	if result.AccountType != wantType {
		t.Fatalf("AccountType = %q, want %q", result.AccountType, wantType)
	}
	if result.IsBusiness == nil {
		t.Fatal("IsBusiness = nil, want a definite value")
	}
	if *result.IsBusiness != wantBusiness {
		t.Fatalf("IsBusiness = %t, want %t", *result.IsBusiness, wantBusiness)
	}
}

func assertUnknownAccountType(t *testing.T, result *AccountTypeStruct) {
	t.Helper()
	if result.AccountType != AccountTypeUnknown {
		t.Fatalf("AccountType = %q, want %q", result.AccountType, AccountTypeUnknown)
	}
	if result.IsBusiness != nil {
		t.Fatalf("IsBusiness = %v, want nil for unknown account type", *result.IsBusiness)
	}
}
