package concept

import (
	"errors"
	"testing"
)

func comptimeControlFlowModule(body string) string {
	return "module ComptimeFlow;\nprofile Core;\n\n" + body
}

func TestComptimeControlFlowEvaluates(t *testing.T) {
	t.Parallel()
	source := comptimeControlFlowModule(`enum Shape
{
    Square(int side),
    Rect(int width, int height),
}

comptime int Area(Shape shape)
{
    match (shape)
    {
        Shape::Square(side) => { return side * side; }
        Shape::Rect(width, height) => { return width * height; }
    }
}

comptime int Sides(Shape shape)
{
    int sides = match (shape)
    {
        Shape::Square(side) => 1,
        Shape::Rect(width, height) => 2,
    };
    return sides;
}

comptime int Mixed(int n)
{
    int total = 0;
    for (i in 0..n)
    {
        if (i % 2 == 0)
        {
            total = total + i;
        }
        else
        {
            total = total + 1;
        }
    }
    if (total > 100)
    {
        return 0;
    }
    return total;
}

comptime int FirstOver(int<array>[5] values, int limit)
{
    for (value in values)
    {
        if (value > limit) { return value; }
    }
    return -1;
}

comptime int Countdown()
{
    int total = 0;
    for (i in 10..0 descend 3)
    {
        total = total * 100 + i;
    }
    return total;
}

comptime int Stepped()
{
    int total = 0;
    for (i in 0..10 step 4)
    {
        total = total + i;
    }
    return total;
}

comptime string Describe(Shape shape)
{
    string text = "shape";
    match (shape)
    {
        Shape::Square(side) => { text = text + " square"; }
        Shape::Rect(width, height) => { text = text + " rect"; }
    }
    return text + "!";
}

comptime string Joined = "a" + "b" + "c";

static_assert(Area(Shape::Rect(3, 4)) == 12, "match statement with payload bindings");
static_assert(Sides(Shape::Square(5)) == 1, "match expression");
static_assert(Mixed(6) == 9, "for over a range with if/else");
static_assert(FirstOver([1, 4, 9, 2, 12], 5) == 9, "return from inside for");
static_assert(FirstOver([1, 2, 3, 4, 5], 9) == -1, "for runs to completion");
static_assert(Countdown() == 10070401, "descend visits 10, 7, 4, 1");
static_assert(Stepped() == 12, "step visits 0, 4, 8");
static_assert(Describe(Shape::Rect(1, 2)) == "shape rect!", "concatenation in a function");
static_assert(Joined == "abc", "concatenation in a declaration");
`)
	if _, err := Parse("flow.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestComptimeControlFlowIsChecked(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, body, code string
	}{
		{"failing assertion", `comptime int Two()
{
    int n = 0;
    for (i in 0..2) { n = n + 1; }
    return n;
}
static_assert(Two() == 3, "wrong");
`, "CV4207"},
		{"loop bound", `comptime int Long()
{
    int n = 0;
    for (i in 0..1000) { n = n + 1; }
    return n;
}
comptime int L = Long();
`, "CV4206"},
		{"range direction", `comptime int Backwards(int start)
{
    int n = 0;
    for (i in start..0) { n = n + 1; }
    return n;
}
comptime int B = Backwards(5);
`, "RANGE_STEP_INVALID"},
		{"if condition", `comptime int Bad(int n)
{
    if (n) { return 1; }
    return 0;
}
`, "CV4186"},
		{"runtime concatenation", `int Runtime()
{
    string s = "a" + "b";
    return 0;
}
`, "STRING_CONCAT_RUNTIME"},
		{"literal range direction", `comptime int Literal()
{
    int n = 0;
    for (i in 5..0) { n = n + 1; }
    return n;
}
`, "RANGE_DIRECTION_INVALID"},
		{"span iteration", `comptime int Spans(int<array>[3] values)
{
    int n = 0;
    for (value in Span(values)) { n = n + value; }
    return n;
}
`, "FOREACH_ITERATOR_INVALID"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse("flow.concept", comptimeControlFlowModule(tc.body))
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
			if diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %s: %s", tc.code, diagnostic.Code, diagnostic.Message)
			}
		})
	}
}
