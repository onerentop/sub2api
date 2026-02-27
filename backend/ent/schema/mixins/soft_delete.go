// Package mixins 提供 Ent schema 的可复用混入组件。
// 包括时间戳混入、软删除混入等通用功能。
package mixins

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SoftDeleteMixin 实现基于 deleted_at 时间戳的软删除功能。
//
// 软删除特性：
//   - 删除操作不会真正删除数据库记录，而是设置 deleted_at 时间戳
//   - 所有查询默认自动过滤 deleted_at IS NULL，只返回"未删除"的记录
//   - 通过 SkipSoftDelete(ctx) 可以绕过软删除过滤器，查询或真正删除记录
//
// 实现原理：
//   - 使用 Ent 的 Interceptor 拦截所有查询，自动添加 deleted_at IS NULL 条件
//   - 使用 Ent 的 Hook 拦截删除操作，将 DELETE 转换为 UPDATE SET deleted_at = NOW()
//
// 使用示例：
//
//	func (User) Mixin() []ent.Mixin {
//	    return []ent.Mixin{
//	        mixins.SoftDeleteMixin{},
//	    }
//	}
type SoftDeleteMixin struct {
	mixin.Schema
}

// Fields 定义软删除所需的字段。
// deleted_at 字段：
//   - 类型为 TIMESTAMPTZ，精确记录删除时间
//   - Optional 和 Nillable 确保新记录时该字段为 NULL
//   - NULL 表示记录未被删除，非 NULL 表示已软删除
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

// softDeleteKey 是用于在 context 中标记跳过软删除的键类型。
// 使用空结构体作为键可以避免与其他包的键冲突。
type softDeleteKey struct{}

// SkipSoftDelete 返回一个新的 context，用于跳过软删除的拦截器和变更器。
//
// 使用场景：
//   - 查询已软删除的记录（如管理员查看回收站）
//   - 执行真正的物理删除（如彻底清理数据）
//   - 恢复软删除的记录
//
// 示例：
//
//	// 查询包含已删除记录的所有用户
//	users, err := client.User.Query().All(mixins.SkipSoftDelete(ctx))
//
//	// 真正删除记录
//	client.User.DeleteOneID(id).Exec(mixins.SkipSoftDelete(ctx))
func SkipSoftDelete(parent context.Context) context.Context {
	return context.WithValue(parent, softDeleteKey{}, true)
}

// softDeleteQuery 是用于软删除拦截器的通用接口
type softDeleteQuery interface {
	WhereP(...func(*sql.Selector))
}

// Interceptors 返回查询拦截器列表。
// 拦截器会自动为所有查询添加 deleted_at IS NULL 条件，
// 确保软删除的记录不会出现在普通查询结果中。
//
// 兼容 Ent v0.14+：
// v0.14 移除了 *XxxQuery 上的 WhereP 方法（改为通过 intercept.Query 包装器提供），
// 因此除了检查 softDeleteQuery 接口外，还通过反射调用 Where 方法来添加谓词。
func (d SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		ent.InterceptFunc(func(next ent.Querier) ent.Querier {
			return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
				// 检查是否需要跳过软删除过滤
				if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
					return next.Query(ctx, query)
				}
				// 为查询添加 deleted_at IS NULL 条件
				if q, ok := query.(softDeleteQuery); ok {
					// Ent 旧版本：Query 类型直接实现 WhereP
					d.applyPredicate(q)
				} else {
					// Ent v0.14+：Query 类型不再直接实现 WhereP，
					// 通过反射调用 Where 方法（Where 接受的 predicate.X 底层类型为 func(*sql.Selector)）
					d.applyPredicateReflect(query)
				}
				return next.Query(ctx, query)
			})
		}),
	}
}

// Hooks 返回变更钩子列表。
// 钩子会拦截 DELETE 操作，将其转换为 UPDATE SET deleted_at = NOW()。
// 这样删除操作实际上只是标记记录为已删除，而不是真正删除。
func (d SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				// 只处理删除操作
				if m.Op() != ent.OpDelete && m.Op() != ent.OpDeleteOne {
					return next.Mutate(ctx, m)
				}
				// 检查是否需要执行真正的删除
				if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
					return next.Mutate(ctx, m)
				}
				// 类型断言，获取 mutation 的扩展接口
				mx, ok := m.(interface {
					SetOp(ent.Op)
					SetDeletedAt(time.Time)
					WhereP(...func(*sql.Selector))
				})
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				// 添加软删除过滤条件，确保不会影响已删除的记录
				d.applyPredicate(mx)
				// 将 DELETE 操作转换为 UPDATE 操作
				mx.SetOp(ent.OpUpdate)
				// 设置删除时间为当前时间
				mx.SetDeletedAt(time.Now())
				return mutateWithClient(ctx, m, next)
			})
		},
	}
}

// applyPredicate 为查询添加 deleted_at IS NULL 条件。
// 这是软删除过滤的核心实现（用于实现 WhereP 接口的类型）。
func (d SoftDeleteMixin) applyPredicate(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.FieldIsNull(d.Fields()[0].Descriptor().Name),
	)
}

// applyPredicateReflect 通过反射为查询添加 deleted_at IS NULL 条件。
// 适用于 Ent v0.14+ 中 *XxxQuery 不再直接实现 WhereP 的情况。
// 通过反射调用 Where 方法，传入 predicate 函数（底层类型为 func(*sql.Selector)）。
func (d SoftDeleteMixin) applyPredicateReflect(query ent.Query) {
	qv := reflect.ValueOf(query)
	whereMethod := qv.MethodByName("Where")
	if !whereMethod.IsValid() {
		return
	}
	// 构建 deleted_at IS NULL 谓词
	pred := sql.FieldIsNull(d.Fields()[0].Descriptor().Name)
	// Where 方法接受可变参数 predicate.X（底层类型为 func(*sql.Selector)），
	// 通过反射构建正确类型的参数
	methodType := whereMethod.Type()
	if methodType.NumIn() < 1 || !methodType.IsVariadic() {
		return
	}
	// 获取可变参数的元素类型（如 predicate.Proxy）
	elemType := methodType.In(0).Elem()
	// 将 func(*sql.Selector) 转换为目标谓词类型
	predValue := reflect.MakeFunc(elemType, func(args []reflect.Value) []reflect.Value {
		sel := args[0].Interface().(*sql.Selector)
		pred(sel)
		return nil
	})
	// 调用 Where(predValue)
	whereMethod.Call([]reflect.Value{predValue})
}

func mutateWithClient(ctx context.Context, m ent.Mutation, fallback ent.Mutator) (ent.Value, error) {
	clientMethod := reflect.ValueOf(m).MethodByName("Client")
	if !clientMethod.IsValid() || clientMethod.Type().NumIn() != 0 || clientMethod.Type().NumOut() != 1 {
		return nil, fmt.Errorf("soft delete: mutation client method not found for %T", m)
	}
	client := clientMethod.Call(nil)[0]
	mutateMethod := client.MethodByName("Mutate")
	if !mutateMethod.IsValid() {
		return nil, fmt.Errorf("soft delete: mutation client missing Mutate for %T", m)
	}
	if mutateMethod.Type().NumIn() != 2 || mutateMethod.Type().NumOut() != 2 {
		return nil, fmt.Errorf("soft delete: mutation client signature mismatch for %T", m)
	}

	results := mutateMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(m)})
	value := results[0].Interface()
	var err error
	if !results[1].IsNil() {
		errValue := results[1].Interface()
		typedErr, ok := errValue.(error)
		if !ok {
			return nil, fmt.Errorf("soft delete: unexpected error type %T for %T", errValue, m)
		}
		err = typedErr
	}
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("soft delete: mutation client returned nil for %T", m)
	}
	v, ok := value.(ent.Value)
	if !ok {
		return nil, fmt.Errorf("soft delete: unexpected value type %T for %T", value, m)
	}
	return v, nil
}
