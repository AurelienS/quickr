package reserved

import "testing"

func TestIsReservedAlias(t *testing.T) {
    reservedCases := []string{"admin", "stats", "LOGIN", "/magic/", "robots.txt", "favicon.ico"}
    for _, a := range reservedCases {
        if !IsReservedAlias(a) {
            t.Errorf("expected reserved: %q", a)
        }
    }
    freeCases := []string{"my-alias", "foo", "bar"}
    for _, a := range freeCases {
        if IsReservedAlias(a) {
            t.Errorf("expected not reserved: %q", a)
        }
    }
}


