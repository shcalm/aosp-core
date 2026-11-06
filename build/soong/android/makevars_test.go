package android

import (
	"regexp"
	"testing"
)

func TestDistFilesInGenerateAndroidBuildActions(t *testing.T) {
	result := GroupFixturePreparers(
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("my_module_type", newDistFileModule)
			ctx.RegisterParallelSingletonType("my_singleton", newDistFileSingleton)
		}),
		FixtureModifyConfig(SetKatiEnabledForTests),
		PrepareForTestWithMakevars,
	).RunTestWithBp(t, `
	my_module_type {
		name: "foo",
	}
	`)

	lateContents := string(result.SingletonForTests(t, "makevars").Singleton().(*makeVarsSingleton).lateForTesting)
	matched, err := regexp.MatchString(`call dist-for-goals,my_goal,.*/my_file.txt:my_file.txt\)`, lateContents)
	if err != nil || !matched {
		t.Fatalf("Expected a dist of my_file.txt, but got: %s", lateContents)
	}
	matched, err = regexp.MatchString(`call dist-for-goals,my_singleton_goal,.*/my_singleton_file.txt:my_singleton_file.txt\)`, lateContents)
	if err != nil || !matched {
		t.Fatalf("Expected a dist of my_singleton_file.txt, but got: %s", lateContents)
	}
}

type distFileModule struct {
	ModuleBase
}

func newDistFileModule() Module {
	m := &distFileModule{}
	InitAndroidModule(m)
	return m
}

func (m *distFileModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	out := PathForModuleOut(ctx, "my_file.txt")
	WriteFileRule(ctx, out, "Hello, world!")
	ctx.DistForGoal("my_goal", out)
}

type distFileSingleton struct {
}

func newDistFileSingleton() Singleton {
	return &distFileSingleton{}
}

func (d *distFileSingleton) GenerateBuildActions(ctx SingletonContext) {
	out := PathForOutput(ctx, "my_singleton_file.txt")
	WriteFileRule(ctx, out, "Hello, world!")
	ctx.DistForGoal("my_singleton_goal", out)
}
