// F02-TEST-09 (F02-AC-06, BR-12): a Meals item of at most 1,500.00.
func TestF02TEST09MealsLimit(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{"MEALS 1500.00", []LineItemInput{item("MEALS", 150000)}, nil},
		{"MEALS 1500.01", []LineItemInput{item("MEALS", 150001)}, []want{{"items[0].amount", "BR-12"}}},
		{"TRAVEL 1500.01", []LineItemInput{item("TRAVEL", 150001)}, nil},
	})
}

// ... runner from the same file, lines 72-90 ...

// runRuleCases creates a claim for each case through claim.Service against PostgreSQL.
func runRuleCases(t *testing.T, cases []ruleCase) {
	t.Helper()

	pool := pgtest.New(t)
	service, actor := ruleService(t, pool)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := Input{Title: "Title", Purpose: "Purpose", Items: test.items}

			_, err := service.Create(t.Context(), actor, input, true)
			if got := violations(t, err); !sameViolations(got, test.want) {
				t.Fatalf("violations = %v, want %v", got, test.want)
			}
		})
	}
}
