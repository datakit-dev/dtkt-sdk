package resource

import "testing"

func TestEmptyName(t *testing.T) {
	empty := EmptyName()
	if empty.String() != "" {
		t.Fatalf("expected empty name to be empty string, got: %s", empty.String())
	}
}

func TestNameType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTyp NameType
		wantOk  bool
	}{
		{
			name:    "Valid user",
			input:   "users/alice",
			wantTyp: User,
			wantOk:  true,
		},
		{
			name:    "Valid organization",
			input:   "organizations/acme",
			wantTyp: Organization,
			wantOk:  true,
		},
		{
			name:    "Valid connection",
			input:   "users/alice/connections/postgres",
			wantTyp: Connection,
			wantOk:  true,
		},
		{
			name:    "Valid type",
			input:   "types/dtkt.protoui.v1beta1.Label",
			wantTyp: Type,
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := NewName(tt.input)
			t.Log(tt.wantTyp.Pattern(), name)

			typ, ok := name.Type()
			if ok != tt.wantOk {
				t.Errorf("Type() ok = %v, want %v", ok, tt.wantOk)
			}
			if typ != tt.wantTyp {
				t.Errorf("Type() type = %v, want %v", typ, tt.wantTyp)
			}
		})
	}
}

// TestNewName tests parsing string paths into Name
func TestNewName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Name
	}{
		{
			name:  "User path",
			input: "users/alice",
			want:  Name{{"users", "alice"}},
		},
		{
			name:  "User with connection",
			input: "users/alice/connections/postgres",
			want:  Name{{"users", "alice"}, {"connections", "postgres"}},
		},
		{
			name:  "Org with integration",
			input: "organizations/acme/integrations/bigquery",
			want:  Name{{"organizations", "acme"}, {"integrations", "bigquery"}},
		},
		{
			name:  "Deep nesting",
			input: "users/bob/connections/db/types/record",
			want:  Name{{"users", "bob"}, {"connections", "db"}, {"types", "record"}},
		},
		{
			name:  "Empty string",
			input: "",
			want:  Name{{"", ""}},
		},
		{
			name:  "With leading slash",
			input: "/users/alice",
			want:  Name{{"users", "alice"}},
		},
		{
			name:  "With trailing slash",
			input: "users/alice/",
			want:  Name{{"users", "alice"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewName(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("NewName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestNameOf_API tests the NameOf API for creating bare resource segments
func TestNameOf_API(t *testing.T) {
	tests := []struct {
		name      string
		typ       NameType
		values    []string
		want      string
		wantError bool
	}{
		{
			name:   "User",
			typ:    User,
			values: []string{"alice"},
			want:   "users/alice",
		},
		{
			name:   "Organization",
			typ:    Organization,
			values: []string{"acme"},
			want:   "organizations/acme",
		},
		{
			name:   "Connection - bare",
			typ:    Connection,
			values: []string{"postgres"},
			want:   "connections/postgres",
		},
		{
			name:   "Integration - bare",
			typ:    Integration,
			values: []string{"bigquery"},
			want:   "integrations/bigquery",
		},
		{
			name:   "Deployment",
			typ:    Deployment,
			values: []string{"prod"},
			want:   "deployments/prod",
		},
		{
			name:   "Flow",
			typ:    Flow,
			values: []string{"etl-pipeline"},
			want:   "flows/etl-pipeline",
		},
		{
			name:   "Automation",
			typ:    Automation,
			values: []string{"daily-sync"},
			want:   "automations/daily-sync",
		},
		{
			name:   "Type",
			typ:    Type,
			values: []string{"record"},
			want:   "types/record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.New(tt.values[0])
			if got.String() != tt.want {
				t.Errorf("NameOf() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

// TestNameOf_Hierarchical tests building hierarchical names using Append
func TestNameOf_Hierarchical(t *testing.T) {
	tests := []struct {
		name string
		fn   func() Name
		want string
	}{
		{
			name: "User with connection",
			fn: func() Name {
				return User.New("alice").Append(Connection, "postgres")
			},
			want: "users/alice/connections/postgres",
		},
		{
			name: "Organization with integration",
			fn: func() Name {
				return Organization.New("acme").Append(Integration, "bigquery")
			},
			want: "organizations/acme/integrations/bigquery",
		},
		{
			name: "User with deployment",
			fn: func() Name {
				return User.New("bob").Append(Deployment, "prod")
			},
			want: "users/bob/deployments/prod",
		},
		{
			name: "Deep nesting - connection with type",
			fn: func() Name {
				return User.New("alice").Append(Connection, "postgres").Append(Type, "record")
			},
			want: "users/alice/connections/postgres/types/record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got.String() != tt.want {
				t.Errorf("Name.String() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

// TestGetName_API tests the GetName API for parsing resource names
func TestGetName_API(t *testing.T) {
	tests := []struct {
		name      string
		typ       NameType
		input     string
		want      string
		wantPairs []Pair
		wantError bool
	}{
		{
			name:      "User",
			typ:       User,
			input:     "users/alice",
			want:      "users/alice",
			wantPairs: []Pair{{"users", "alice"}},
		},
		{
			name:      "Organization",
			typ:       Organization,
			input:     "organizations/acme",
			want:      "organizations/acme",
			wantPairs: []Pair{{"organizations", "acme"}},
		},
		{
			name:      "Connection - with user",
			typ:       Connection,
			input:     "users/alice/connections/postgres",
			want:      "users/alice/connections/postgres",
			wantPairs: []Pair{{"users", "alice"}, {"connections", "postgres"}},
		},
		{
			name:      "Connection - with org",
			typ:       Connection,
			input:     "organizations/acme/connections/postgres",
			want:      "organizations/acme/connections/postgres",
			wantPairs: []Pair{{"organizations", "acme"}, {"connections", "postgres"}},
		},
		{
			name:      "Integration - with user",
			typ:       Integration,
			input:     "users/bob/integrations/bigquery",
			want:      "users/bob/integrations/bigquery",
			wantPairs: []Pair{{"users", "bob"}, {"integrations", "bigquery"}},
		},
		{
			name:      "Type - system level",
			typ:       Type,
			input:     "types/dtkt.protoui.v1beta1.Label",
			want:      "types/dtkt.protoui.v1beta1.Label",
			wantPairs: []Pair{{"types", "dtkt.protoui.v1beta1.Label"}},
		},
		{
			name:      "Type - nested in connection",
			typ:       Type,
			input:     "users/alice/connections/postgres/types/Record",
			want:      "users/alice/connections/postgres/types/Record",
			wantPairs: []Pair{{"users", "alice"}, {"connections", "postgres"}, {"types", "Record"}},
		},
		{
			name:      "Service - system level",
			typ:       Service,
			input:     "services/MyService",
			want:      "services/MyService",
			wantPairs: []Pair{{"services", "MyService"}},
		},
		{
			name:      "Service - nested in deployment",
			typ:       Service,
			input:     "users/bob/deployments/prod/services/api.v1.UserService",
			want:      "users/bob/deployments/prod/services/api.v1.UserService",
			wantPairs: []Pair{{"users", "bob"}, {"deployments", "prod"}, {"services", "api.v1.UserService"}},
		},
		{
			name:      "Method - system level",
			typ:       Method,
			input:     "methods/GetUser",
			want:      "methods/GetUser",
			wantPairs: []Pair{{"methods", "GetUser"}},
		},
		{
			name:      "Method - nested in connection",
			typ:       Method,
			input:     "organizations/acme/connections/postgres/methods/query",
			want:      "organizations/acme/connections/postgres/methods/query",
			wantPairs: []Pair{{"organizations", "acme"}, {"connections", "postgres"}, {"methods", "query"}},
		},
		{
			name:      "Operation - under automation",
			typ:       Operation,
			input:     "users/alice/automations/daily-sync/operations/transform",
			want:      "users/alice/automations/daily-sync/operations/transform",
			wantPairs: []Pair{{"users", "alice"}, {"automations", "daily-sync"}, {"operations", "transform"}},
		},
		{
			name:      "Operation - under deployment",
			typ:       Operation,
			input:     "organizations/acme/deployments/prod/operations/init",
			want:      "organizations/acme/deployments/prod/operations/init",
			wantPairs: []Pair{{"organizations", "acme"}, {"deployments", "prod"}, {"operations", "init"}},
		},
		{
			name:      "Operation - under integration",
			typ:       Operation,
			input:     "users/bob/integrations/bigquery/operations/sync",
			want:      "users/bob/integrations/bigquery/operations/sync",
			wantPairs: []Pair{{"users", "bob"}, {"integrations", "bigquery"}, {"operations", "sync"}},
		},
		{
			name:      "Invalid format",
			typ:       User,
			input:     "invalid/path/structure",
			wantError: true,
		},
		{
			name:      "Wrong collection name",
			typ:       User,
			input:     "organizations/alice",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.typ.GetName(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("GetNameFor() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			if got.String() != tt.want {
				t.Errorf("GetNameFor().String() = %v, want %v", got.String(), tt.want)
			}

			if len(got) != len(tt.wantPairs) {
				t.Errorf("GetNameFor() has %d pairs, want %d", len(got), len(tt.wantPairs))
				return
			}

			for i, pair := range tt.wantPairs {
				if got[i][0] != pair[0] || got[i][1] != pair[1] {
					t.Errorf("GetNameFor()[%d] = %v, want %v", i, got[i], pair)
				}
			}
		})
	}
}

// TestNameType_Methods tests the NameType convenience methods
func TestNameType_Methods(t *testing.T) {
	// Test IsName
	if !User.IsName("users/alice") {
		t.Error("User.IsName('users/alice') = false, want true")
	}
	if User.IsName("organizations/alice") {
		t.Error("User.IsName('organizations/alice') = true, want false")
	}

	// Test GetName - connections require parent context
	name, err := Connection.GetName("users/alice/connections/postgres")
	if err != nil {
		t.Errorf("Connection.GetName() error = %v", err)
	}
	if name.String() != "users/alice/connections/postgres" {
		t.Errorf("Connection.GetName() = %v, want users/alice/connections/postgres", name.String())
	}

	// Test MustGetName - integrations require parent context
	name = Integration.MustGetName("organizations/acme/integrations/bigquery")
	if name.String() != "organizations/acme/integrations/bigquery" {
		t.Errorf("Integration.MustGetName() = %v, want organizations/acme/integrations/bigquery", name.String())
	}
}

// TestName_Append tests building names incrementally
func TestName_Append(t *testing.T) {
	// Build a full resource path by appending
	name := User.New("alice")
	t.Log(name)
	name = name.Append(Connection, "postgres")
	t.Log(name)
	name = name.Append(Type, "record")
	t.Log(name)

	want := "users/alice/connections/postgres/types/record"
	if name.String() != want {
		t.Fatalf("Name.Append() = %v, want %v", name.String(), want)
	}

	// Check individual components
	if len(name) != 3 {
		t.Fatalf("Name length = %d, want 3", len(name))
	}
	if name[0][0] != "users" || name[0][1] != "alice" {
		t.Fatalf("Name[0] = %#v, want [user alice]", name[0])
	}
	if name[1][0] != "connections" || name[1][1] != "postgres" {
		t.Fatalf("Name[1] = %#v, want [connection postgres]", name[1])
	}
	if name[2][0] != "types" || name[2][1] != "record" {
		t.Fatalf("Name[2] = %#v, want [type record]", name[2])
	}
}

// TestName_Parent tests the Parent method for hierarchical navigation
func TestNameParent(t *testing.T) {
	// Test connection parent
	connName := "users/jordan/connections/petstore"
	n, err := Connection.GetName(connName)
	if err != nil {
		t.Fatalf("unexpected error getting name: %s", err)
	}

	parent := n.Parent()
	expectedParent := "users/jordan"
	if parent.String() != expectedParent {
		t.Fatalf("expected parent %s, got %s", expectedParent, parent.String())
	}

	// Test type parent (deeper hierarchy)
	typeName := "users/jordan/connections/petstore/types/google.protobuf.MethodDescriptorProto"
	n, err = Type.GetName(typeName)
	if err != nil {
		t.Fatalf("unexpected error getting name: %s", err)
	}

	parent = n.Parent()
	expectedParent = "users/jordan/connections/petstore"
	if parent.String() != expectedParent {
		t.Fatalf("expected parent %s, got %s", expectedParent, parent.String())
	}

	// Test root resources have no parent
	userOnly := User.New("alice")
	parent = userOnly.Parent()
	if parent.String() != "" {
		t.Errorf("root resource parent = %v, want empty", parent.String())
	}
}

// TestName_FirstLast tests the First and Last methods
func TestName_FirstLast(t *testing.T) {
	name := User.New("alice")
	name = name.Append(Connection, "postgres")
	name = name.Append(Type, "record")

	// Test First
	first := name.First()
	if first.String() != "users/alice" {
		t.Errorf("Name.First() = %v, want users/alice", first.String())
	}

	// Test Last
	last := name.Last()
	if last.String() != "types/record" {
		t.Errorf("Name.Last() = %v, want types/record", last.String())
	}
}

// TestName_Short tests the Short method
func TestName_Short(t *testing.T) {
	tests := []struct {
		name string
		n    Name
		want string
	}{
		{
			name: "Single component",
			n:    User.New("alice"),
			want: "alice",
		},
		{
			name: "Multiple components",
			n:    User.New("alice").Append(Connection, "postgres"),
			want: "postgres",
		},
		{
			name: "Empty name",
			n:    EmptyName(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.n.Short()
			if got != tt.want {
				t.Errorf("Name.Short() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestName_Equal tests the Equal method
func TestName_Equal(t *testing.T) {
	name1 := User.New("alice")
	name2 := User.New("alice")
	name3 := User.New("bob")
	name4 := Organization.New("alice")

	if !name1.Equal(name2) {
		t.Error("Equal names not equal")
	}
	if name1.Equal(name3) {
		t.Error("Different values equal")
	}
	if name1.Equal(name4) {
		t.Error("Different types equal")
	}
}

// TestIsName_API tests the IsName function
func TestIsName_API(t *testing.T) {
	tests := []struct {
		name  string
		typ   NameType
		input string
		want  bool
	}{
		{"Valid user", User, "users/alice", true},
		{"Valid org", Organization, "organizations/acme", true},
		{"Valid connection", Connection, "users/alice/connections/postgres", true},
		{"Invalid - wrong collection", User, "organizations/alice", false},
		{"Invalid - malformed", User, "users/alice/extra", false},
		{"Invalid - empty", User, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.IsName(tt.input)
			if got != tt.want {
				t.Errorf("IsName(%v, %v) = %v, want %v", tt.typ, tt.input, got, tt.want)
			}
		})
	}
}

// TestTemplatePatternAlignment validates that template parameter names match pattern capture groups
// This test ensures that when you add a new resource type, the template like "users/{user}" has
// its parameter name "user" matching the pattern's capture group name "(?P<user>...)"
func TestTemplatePatternAlignment(t *testing.T) {
	types := []NameType{User, Organization, Integration, Deployment, Connection, Flow, Automation, Type, Service, Method, Operation}

	for _, typ := range types {
		t.Run(string(typ), func(t *testing.T) {
			// Build a test name with a dummy value
			testName := typ.New("test")
			if len(testName) == 0 {
				t.Fatalf("NameOf(%s, \"test\") returned empty Name", typ)
			}

			// Get the parameter name extracted from the template
			templateParam := testName[0][0]

			// Build the full name for validation (needs a parent context for most types)
			var fullPath string
			switch typ {
			case User, Organization:
				fullPath = testName.String()
			case Type, Service, Method:
				// These can be system-level
				fullPath = testName.String()
			case Operation:
				// Operation requires a specific parent (Automation, Deployment, or Integration)
				fullPath = User.New("testuser").Append(Automation, "testauto").Append(typ, "test").String()
			default:
				// Most resources need a parent (user or org)
				fullPath = User.New("testuser").Append(typ, "test").String()
			}

			// Parse it back and verify we get the same parameter name
			parsedName, err := typ.GetName(fullPath)
			if err != nil {
				t.Fatalf("GetNameFor(%s, %q) error: %v", typ, fullPath, err)
			}

			// Find the parameter in the parsed name
			found := false
			for _, pair := range parsedName {
				if pair[0] == templateParam {
					found = true
					if pair[1] != "test" {
						t.Errorf("Pattern extracted wrong value: got %q, want \"test\"", pair[1])
					}
					break
				}
			}

			if !found {
				t.Errorf("Template param %q not found in parsed name %v - template/pattern mismatch!",
					templateParam, parsedName)
			}
		})
	}
}
