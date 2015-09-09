package worker

import "gitlab.com/cretz/fusty/model"

// We don't return anything because this is expected to be invoked asynchronously without regard for the response
func do(exec *model.Execution) {
	panic("TODO")
}
