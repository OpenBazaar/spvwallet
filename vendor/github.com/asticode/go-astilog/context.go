package astilog

import "context"

const contextKeyFields = "astilog.fields"

func fieldsFromContext(ctx context.Context) Fields {
	v, ok := ctx.Value(contextKeyFields).(Fields)
	if !ok {
		return nil
	}
	return v
}

func ContextWithField(parent context.Context, k string, v interface{}) context.Context {
	return ContextWithFields(parent, Fields{k: v})
}

func ContextWithFields(parent context.Context, fs Fields) context.Context {
	cfs := fieldsFromContext(parent)
	if cfs == nil {
		cfs = make(Fields)
	}
	for k, v := range fs {
		cfs[k] = v
	}
	return context.WithValue(parent, contextKeyFields, cfs)
}
