//go:build bindings

package main

func newApplication() (*ApplicationRuntime, error) {
	return &ApplicationRuntime{
		bindings: []interface{}{
			&ModeController{},
			&NormalController{},
			&RecoveryController{},
		},
	}, nil
}
