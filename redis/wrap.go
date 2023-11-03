package redis

import (
	"context"
	"time"

	rdsDrv "github.com/redis/go-redis/v9"
)

func (i *instance) Pipeline() rdsDrv.Pipeliner {
	return i.GetProxy().Pipeline()
}
func (i *instance) Pipelined(ctx context.Context, fn func(rdsDrv.Pipeliner) error) ([]rdsDrv.Cmder, error) {
	return i.GetProxy().Pipelined(ctx, fn)
}
func (i *instance) TxPipelined(ctx context.Context, fn func(rdsDrv.Pipeliner) error) ([]rdsDrv.Cmder, error) {
	return i.GetProxy().TxPipelined(ctx, fn)
}
func (i *instance) TxPipeline() rdsDrv.Pipeliner {
	return i.GetProxy().TxPipeline()
}
func (i *instance) Command(ctx context.Context) *rdsDrv.CommandsInfoCmd {
	return i.GetProxy().Command(ctx)
}
func (i *instance) CommandList(ctx context.Context, filter *rdsDrv.FilterBy) *rdsDrv.StringSliceCmd {
	return i.GetProxy().CommandList(ctx, filter)
}
func (i *instance) CommandGetKeys(ctx context.Context, commands ...any) *rdsDrv.StringSliceCmd {
	return i.GetProxy().CommandGetKeys(ctx, commands...)
}
func (i *instance) CommandGetKeysAndFlags(ctx context.Context, commands ...any) *rdsDrv.KeyFlagsCmd {
	return i.GetProxy().CommandGetKeysAndFlags(ctx, commands...)
}
func (i *instance) ClientGetName(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().ClientGetName(ctx)
}
func (i *instance) Echo(ctx context.Context, message any) *rdsDrv.StringCmd {
	return i.GetProxy().Echo(ctx, message)
}
func (i *instance) Ping(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().Ping(ctx)
}
func (i *instance) Quit(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().Quit(ctx)
}
func (i *instance) Del(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().Del(ctx, keys...)
}
func (i *instance) Unlink(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().Unlink(ctx, keys...)
}
func (i *instance) Dump(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().Dump(ctx, key)
}
func (i *instance) Exists(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().Exists(ctx, keys...)
}
func (i *instance) Expire(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().Expire(ctx, key, expiration)
}
func (i *instance) ExpireAt(ctx context.Context, key string, tm time.Time) *rdsDrv.BoolCmd {
	return i.GetProxy().ExpireAt(ctx, key, tm)
}
func (i *instance) ExpireTime(ctx context.Context, key string) *rdsDrv.DurationCmd {
	return i.GetProxy().ExpireTime(ctx, key)
}
func (i *instance) ExpireNX(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().ExpireNX(ctx, key, expiration)
}
func (i *instance) ExpireXX(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().ExpireXX(ctx, key, expiration)
}
func (i *instance) ExpireGT(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().ExpireGT(ctx, key, expiration)
}
func (i *instance) ExpireLT(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().ExpireLT(ctx, key, expiration)
}
func (i *instance) Keys(ctx context.Context, pattern string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().Keys(ctx, pattern)
}
func (i *instance) Migrate(ctx context.Context, host, port, key string, db int,
	timeout time.Duration) *rdsDrv.StatusCmd {
	return i.GetProxy().Migrate(ctx, host, port, key, db, timeout)
}
func (i *instance) Move(ctx context.Context, key string, db int) *rdsDrv.BoolCmd {
	return i.GetProxy().Move(ctx, key, db)
}
func (i *instance) ObjectRefCount(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().ObjectRefCount(ctx, key)
}
func (i *instance) ObjectEncoding(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().ObjectEncoding(ctx, key)
}
func (i *instance) ObjectIdleTime(ctx context.Context, key string) *rdsDrv.DurationCmd {
	return i.GetProxy().ObjectIdleTime(ctx, key)
}
func (i *instance) Persist(ctx context.Context, key string) *rdsDrv.BoolCmd {
	return i.GetProxy().Persist(ctx, key)
}
func (i *instance) PExpire(ctx context.Context, key string, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().PExpire(ctx, key, expiration)
}
func (i *instance) PExpireAt(ctx context.Context, key string, tm time.Time) *rdsDrv.BoolCmd {
	return i.GetProxy().PExpireAt(ctx, key, tm)
}
func (i *instance) PExpireTime(ctx context.Context, key string) *rdsDrv.DurationCmd {
	return i.GetProxy().PExpireTime(ctx, key)
}
func (i *instance) PTTL(ctx context.Context, key string) *rdsDrv.DurationCmd {
	return i.GetProxy().PTTL(ctx, key)
}
func (i *instance) RandomKey(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().RandomKey(ctx)
}
func (i *instance) Rename(ctx context.Context, key, newkey string) *rdsDrv.StatusCmd {
	return i.GetProxy().Rename(ctx, key, newkey)
}
func (i *instance) RenameNX(ctx context.Context, key, newkey string) *rdsDrv.BoolCmd {
	return i.GetProxy().RenameNX(ctx, key, newkey)
}
func (i *instance) Restore(ctx context.Context, key string, ttl time.Duration, value string) *rdsDrv.StatusCmd {
	return i.GetProxy().Restore(ctx, key, ttl, value)
}
func (i *instance) RestoreReplace(ctx context.Context, key string, ttl time.Duration, value string) *rdsDrv.StatusCmd {
	return i.GetProxy().RestoreReplace(ctx, key, ttl, value)
}
func (i *instance) Sort(ctx context.Context, key string, sort *rdsDrv.Sort) *rdsDrv.StringSliceCmd {
	return i.GetProxy().Sort(ctx, key, sort)
}
func (i *instance) SortRO(ctx context.Context, key string, sort *rdsDrv.Sort) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SortRO(ctx, key, sort)
}
func (i *instance) SortStore(ctx context.Context, key, store string, sort *rdsDrv.Sort) *rdsDrv.IntCmd {
	return i.GetProxy().SortStore(ctx, key, store, sort)
}
func (i *instance) SortInterfaces(ctx context.Context, key string, sort *rdsDrv.Sort) *rdsDrv.SliceCmd {
	return i.GetProxy().SortInterfaces(ctx, key, sort)
}
func (i *instance) Touch(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().Touch(ctx, keys...)
}
func (i *instance) TTL(ctx context.Context, key string) *rdsDrv.DurationCmd {
	return i.GetProxy().TTL(ctx, key)
}
func (i *instance) Type(ctx context.Context, key string) *rdsDrv.StatusCmd {
	return i.GetProxy().Type(ctx, key)
}
func (i *instance) Append(ctx context.Context, key, value string) *rdsDrv.IntCmd {
	return i.GetProxy().Append(ctx, key, value)
}
func (i *instance) Decr(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().Decr(ctx, key)
}
func (i *instance) DecrBy(ctx context.Context, key string, decrement int64) *rdsDrv.IntCmd {
	return i.GetProxy().DecrBy(ctx, key, decrement)
}
func (i *instance) Get(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().Get(ctx, key)
}
func (i *instance) GetRange(ctx context.Context, key string, start, end int64) *rdsDrv.StringCmd {
	return i.GetProxy().GetRange(ctx, key, start, end)
}
func (i *instance) GetSet(ctx context.Context, key string, value any) *rdsDrv.StringCmd {
	return i.GetProxy().GetSet(ctx, key, value)
}
func (i *instance) GetEx(ctx context.Context, key string, expiration time.Duration) *rdsDrv.StringCmd {
	return i.GetProxy().GetEx(ctx, key, expiration)
}
func (i *instance) GetDel(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().GetDel(ctx, key)
}
func (i *instance) Incr(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().Incr(ctx, key)
}
func (i *instance) IncrBy(ctx context.Context, key string, value int64) *rdsDrv.IntCmd {
	return i.GetProxy().IncrBy(ctx, key, value)
}
func (i *instance) IncrByFloat(ctx context.Context, key string, value float64) *rdsDrv.FloatCmd {
	return i.GetProxy().IncrByFloat(ctx, key, value)
}
func (i *instance) MGet(ctx context.Context, keys ...string) *rdsDrv.SliceCmd {
	return i.GetProxy().MGet(ctx, keys...)
}
func (i *instance) MSet(ctx context.Context, values ...any) *rdsDrv.StatusCmd {
	return i.GetProxy().MSet(ctx, values...)
}
func (i *instance) MSetNX(ctx context.Context, values ...any) *rdsDrv.BoolCmd {
	return i.GetProxy().MSetNX(ctx, values...)
}
func (i *instance) Set(ctx context.Context, key string, value any, expiration time.Duration) *rdsDrv.StatusCmd {
	return i.GetProxy().Set(ctx, key, value, expiration)
}
func (i *instance) SetArgs(ctx context.Context, key string, value any, a rdsDrv.SetArgs) *rdsDrv.StatusCmd {
	return i.GetProxy().SetArgs(ctx, key, value, a)
}
func (i *instance) SetEx(ctx context.Context, key string, value any,
	expiration time.Duration) *rdsDrv.StatusCmd {
	return i.GetProxy().SetEx(ctx, key, value, expiration)
}
func (i *instance) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().SetNX(ctx, key, value, expiration)
}
func (i *instance) SetXX(ctx context.Context, key string, value any, expiration time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().SetXX(ctx, key, value, expiration)
}
func (i *instance) SetRange(ctx context.Context, key string, offset int64, value string) *rdsDrv.IntCmd {
	return i.GetProxy().SetRange(ctx, key, offset, value)
}
func (i *instance) StrLen(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().StrLen(ctx, key)
}
func (i *instance) Copy(ctx context.Context, sourceKey string, destKey string, db int, replace bool) *rdsDrv.IntCmd {
	return i.GetProxy().Copy(ctx, sourceKey, destKey, db, replace)
}
func (i *instance) GetBit(ctx context.Context, key string, offset int64) *rdsDrv.IntCmd {
	return i.GetProxy().GetBit(ctx, key, offset)
}
func (i *instance) SetBit(ctx context.Context, key string, offset int64, value int) *rdsDrv.IntCmd {
	return i.GetProxy().SetBit(ctx, key, offset, value)
}
func (i *instance) BitCount(ctx context.Context, key string, bitCount *rdsDrv.BitCount) *rdsDrv.IntCmd {
	return i.GetProxy().BitCount(ctx, key, bitCount)
}
func (i *instance) BitOpAnd(ctx context.Context, destKey string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().BitOpAnd(ctx, destKey, keys...)
}
func (i *instance) BitOpOr(ctx context.Context, destKey string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().BitOpOr(ctx, destKey, keys...)
}
func (i *instance) BitOpXor(ctx context.Context, destKey string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().BitOpXor(ctx, destKey, keys...)
}
func (i *instance) BitOpNot(ctx context.Context, destKey string, key string) *rdsDrv.IntCmd {
	return i.GetProxy().BitOpNot(ctx, destKey, key)
}
func (i *instance) BitPos(ctx context.Context, key string, bit int64, pos ...int64) *rdsDrv.IntCmd {
	return i.GetProxy().BitPos(ctx, key, bit, pos...)
}
func (i *instance) BitPosSpan(ctx context.Context, key string, bit int8, start, end int64, span string) *rdsDrv.IntCmd {
	return i.GetProxy().BitPosSpan(ctx, key, bit, start, end, span)
}
func (i *instance) BitField(ctx context.Context, key string, args ...any) *rdsDrv.IntSliceCmd {
	return i.GetProxy().BitField(ctx, key, args...)
}
func (i *instance) Scan(ctx context.Context, cursor uint64, match string, count int64) *rdsDrv.ScanCmd {
	return i.GetProxy().Scan(ctx, cursor, match, count)
}
func (i *instance) ScanType(ctx context.Context, cursor uint64, match string, count int64,
	keyType string) *rdsDrv.ScanCmd {
	return i.GetProxy().ScanType(ctx, cursor, match, count, keyType)
}
func (i *instance) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) *rdsDrv.ScanCmd {
	return i.GetProxy().SScan(ctx, key, cursor, match, count)
}
func (i *instance) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *rdsDrv.ScanCmd {
	return i.GetProxy().HScan(ctx, key, cursor, match, count)
}
func (i *instance) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) *rdsDrv.ScanCmd {
	return i.GetProxy().ZScan(ctx, key, cursor, match, count)
}
func (i *instance) HDel(ctx context.Context, key string, fields ...string) *rdsDrv.IntCmd {
	return i.GetProxy().HDel(ctx, key, fields...)
}
func (i *instance) HExists(ctx context.Context, key, field string) *rdsDrv.BoolCmd {
	return i.GetProxy().HExists(ctx, key, field)
}
func (i *instance) HGet(ctx context.Context, key, field string) *rdsDrv.StringCmd {
	return i.GetProxy().HGet(ctx, key, field)
}
func (i *instance) HGetAll(ctx context.Context, key string) *rdsDrv.MapStringStringCmd {
	return i.GetProxy().HGetAll(ctx, key)
}
func (i *instance) HIncrBy(ctx context.Context, key, field string, incr int64) *rdsDrv.IntCmd {
	return i.GetProxy().HIncrBy(ctx, key, field, incr)
}
func (i *instance) HIncrByFloat(ctx context.Context, key, field string, incr float64) *rdsDrv.FloatCmd {
	return i.GetProxy().HIncrByFloat(ctx, key, field, incr)
}
func (i *instance) HKeys(ctx context.Context, key string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().HKeys(ctx, key)
}
func (i *instance) HLen(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().HLen(ctx, key)
}
func (i *instance) HMGet(ctx context.Context, key string, fields ...string) *rdsDrv.SliceCmd {
	return i.GetProxy().HMGet(ctx, key, fields...)
}
func (i *instance) HSet(ctx context.Context, key string, values ...any) *rdsDrv.IntCmd {
	return i.GetProxy().HSet(ctx, key, values...)
}
func (i *instance) HMSet(ctx context.Context, key string, values ...any) *rdsDrv.BoolCmd {
	return i.GetProxy().HMSet(ctx, key, values...)
}
func (i *instance) HSetNX(ctx context.Context, key, field string, value any) *rdsDrv.BoolCmd {
	return i.GetProxy().HSetNX(ctx, key, field, value)
}
func (i *instance) HVals(ctx context.Context, key string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().HVals(ctx, key)
}
func (i *instance) HRandField(ctx context.Context, key string, count int) *rdsDrv.StringSliceCmd {
	return i.GetProxy().HRandField(ctx, key, count)
}
func (i *instance) HRandFieldWithValues(ctx context.Context, key string, count int) *rdsDrv.KeyValueSliceCmd {
	return i.GetProxy().HRandFieldWithValues(ctx, key, count)
}
func (i *instance) BLPop(ctx context.Context, timeout time.Duration, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().BLPop(ctx, timeout, keys...)
}
func (i *instance) BLMPop(ctx context.Context, timeout time.Duration, direction string,
	count int64, keys ...string) *rdsDrv.KeyValuesCmd {
	return i.GetProxy().BLMPop(ctx, timeout, direction, count, keys...)
}
func (i *instance) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().BRPop(ctx, timeout, keys...)
}
func (i *instance) BRPopLPush(ctx context.Context, source, destination string,
	timeout time.Duration) *rdsDrv.StringCmd {
	return i.GetProxy().BRPopLPush(ctx, source, destination, timeout)
}
func (i *instance) LCS(ctx context.Context, q *rdsDrv.LCSQuery) *rdsDrv.LCSCmd {
	return i.GetProxy().LCS(ctx, q)
}
func (i *instance) LIndex(ctx context.Context, key string, index int64) *rdsDrv.StringCmd {
	return i.GetProxy().LIndex(ctx, key, index)
}
func (i *instance) LMPop(ctx context.Context, direction string, count int64, keys ...string) *rdsDrv.KeyValuesCmd {
	return i.GetProxy().LMPop(ctx, direction, count, keys...)
}
func (i *instance) LInsert(ctx context.Context, key, op string, pivot, value any) *rdsDrv.IntCmd {
	return i.GetProxy().LInsert(ctx, key, op, pivot, value)
}
func (i *instance) LInsertBefore(ctx context.Context, key string, pivot, value any) *rdsDrv.IntCmd {
	return i.GetProxy().LInsertBefore(ctx, key, pivot, value)
}
func (i *instance) LInsertAfter(ctx context.Context, key string, pivot, value any) *rdsDrv.IntCmd {
	return i.GetProxy().LInsertAfter(ctx, key, pivot, value)
}
func (i *instance) LLen(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().LLen(ctx, key)
}
func (i *instance) LPop(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().LPop(ctx, key)
}
func (i *instance) LPopCount(ctx context.Context, key string, count int) *rdsDrv.StringSliceCmd {
	return i.GetProxy().LPopCount(ctx, key, count)
}
func (i *instance) LPos(ctx context.Context, key string, value string, args rdsDrv.LPosArgs) *rdsDrv.IntCmd {
	return i.GetProxy().LPos(ctx, key, value, args)
}
func (i *instance) LPosCount(ctx context.Context, key string, value string, count int64,
	args rdsDrv.LPosArgs) *rdsDrv.IntSliceCmd {
	return i.GetProxy().LPosCount(ctx, key, value, count, args)
}
func (i *instance) LPush(ctx context.Context, key string, values ...any) *rdsDrv.IntCmd {
	return i.GetProxy().LPush(ctx, key, values...)
}
func (i *instance) LPushX(ctx context.Context, key string, values ...any) *rdsDrv.IntCmd {
	return i.GetProxy().LPushX(ctx, key, values...)
}
func (i *instance) LRange(ctx context.Context, key string, start, stop int64) *rdsDrv.StringSliceCmd {
	return i.GetProxy().LRange(ctx, key, start, stop)
}
func (i *instance) LRem(ctx context.Context, key string, count int64, value any) *rdsDrv.IntCmd {
	return i.GetProxy().LRem(ctx, key, count, value)
}
func (i *instance) LSet(ctx context.Context, key string, index int64, value any) *rdsDrv.StatusCmd {
	return i.GetProxy().LSet(ctx, key, index, value)
}
func (i *instance) LTrim(ctx context.Context, key string, start, stop int64) *rdsDrv.StatusCmd {
	return i.GetProxy().LTrim(ctx, key, start, stop)
}
func (i *instance) RPop(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().RPop(ctx, key)
}
func (i *instance) RPopCount(ctx context.Context, key string, count int) *rdsDrv.StringSliceCmd {
	return i.GetProxy().RPopCount(ctx, key, count)
}
func (i *instance) RPopLPush(ctx context.Context, source, destination string) *rdsDrv.StringCmd {
	return i.GetProxy().RPopLPush(ctx, source, destination)
}
func (i *instance) RPush(ctx context.Context, key string, values ...any) *rdsDrv.IntCmd {
	return i.GetProxy().RPush(ctx, key, values...)
}
func (i *instance) RPushX(ctx context.Context, key string, values ...any) *rdsDrv.IntCmd {
	return i.GetProxy().RPushX(ctx, key, values...)
}
func (i *instance) LMove(ctx context.Context, source, destination, srcpos, destpos string) *rdsDrv.StringCmd {
	return i.GetProxy().LMove(ctx, source, destination, srcpos, destpos)
}
func (i *instance) BLMove(ctx context.Context, source, destination, srcpos, destpos string,
	timeout time.Duration) *rdsDrv.StringCmd {
	return i.GetProxy().BLMove(ctx, source, destination, srcpos, destpos, timeout)
}
func (i *instance) SAdd(ctx context.Context, key string, members ...any) *rdsDrv.IntCmd {
	return i.GetProxy().SAdd(ctx, key, members...)
}
func (i *instance) SCard(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().SCard(ctx, key)
}
func (i *instance) SDiff(ctx context.Context, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SDiff(ctx, keys...)
}
func (i *instance) SDiffStore(ctx context.Context, destination string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().SDiffStore(ctx, destination, keys...)
}
func (i *instance) SInter(ctx context.Context, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SInter(ctx, keys...)
}
func (i *instance) SInterCard(ctx context.Context, limit int64, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().SInterCard(ctx, limit, keys...)
}
func (i *instance) SInterStore(ctx context.Context, destination string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().SInterStore(ctx, destination, keys...)
}
func (i *instance) SIsMember(ctx context.Context, key string, member any) *rdsDrv.BoolCmd {
	return i.GetProxy().SIsMember(ctx, key, member)
}
func (i *instance) SMIsMember(ctx context.Context, key string, members ...any) *rdsDrv.BoolSliceCmd {
	return i.GetProxy().SMIsMember(ctx, key, members...)
}
func (i *instance) SMembers(ctx context.Context, key string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SMembers(ctx, key)
}
func (i *instance) SMembersMap(ctx context.Context, key string) *rdsDrv.StringStructMapCmd {
	return i.GetProxy().SMembersMap(ctx, key)
}
func (i *instance) SMove(ctx context.Context, source, destination string, member any) *rdsDrv.BoolCmd {
	return i.GetProxy().SMove(ctx, source, destination, member)
}
func (i *instance) SPop(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().SPop(ctx, key)
}
func (i *instance) SPopN(ctx context.Context, key string, count int64) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SPopN(ctx, key, count)
}
func (i *instance) SRandMember(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().SRandMember(ctx, key)
}
func (i *instance) SRandMemberN(ctx context.Context, key string, count int64) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SRandMemberN(ctx, key, count)
}
func (i *instance) SRem(ctx context.Context, key string, members ...any) *rdsDrv.IntCmd {
	return i.GetProxy().SRem(ctx, key, members...)
}
func (i *instance) SUnion(ctx context.Context, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().SUnion(ctx, keys...)
}
func (i *instance) SUnionStore(ctx context.Context, destination string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().SUnionStore(ctx, destination, keys...)
}
func (i *instance) XAdd(ctx context.Context, a *rdsDrv.XAddArgs) *rdsDrv.StringCmd {
	return i.GetProxy().XAdd(ctx, a)
}
func (i *instance) XDel(ctx context.Context, stream string, ids ...string) *rdsDrv.IntCmd {
	return i.GetProxy().XDel(ctx, stream, ids...)
}
func (i *instance) XLen(ctx context.Context, stream string) *rdsDrv.IntCmd {
	return i.GetProxy().XLen(ctx, stream)
}
func (i *instance) XRange(ctx context.Context, stream, start, stop string) *rdsDrv.XMessageSliceCmd {
	return i.GetProxy().XRange(ctx, stream, start, stop)
}
func (i *instance) XRangeN(ctx context.Context, stream, start, stop string, count int64) *rdsDrv.XMessageSliceCmd {
	return i.GetProxy().XRangeN(ctx, stream, start, stop, count)
}
func (i *instance) XRevRange(ctx context.Context, stream string, start, stop string) *rdsDrv.XMessageSliceCmd {
	return i.GetProxy().XRevRange(ctx, stream, start, stop)
}
func (i *instance) XRevRangeN(ctx context.Context, stream string,
	start, stop string, count int64) *rdsDrv.XMessageSliceCmd {
	return i.GetProxy().XRevRangeN(ctx, stream, start, stop, count)
}
func (i *instance) XRead(ctx context.Context, a *rdsDrv.XReadArgs) *rdsDrv.XStreamSliceCmd {
	return i.GetProxy().XRead(ctx, a)
}
func (i *instance) XReadStreams(ctx context.Context, streams ...string) *rdsDrv.XStreamSliceCmd {
	return i.GetProxy().XReadStreams(ctx, streams...)
}
func (i *instance) XGroupCreate(ctx context.Context, stream, group, start string) *rdsDrv.StatusCmd {
	return i.GetProxy().XGroupCreate(ctx, stream, group, start)
}
func (i *instance) XGroupCreateMkStream(ctx context.Context, stream, group, start string) *rdsDrv.StatusCmd {
	return i.GetProxy().XGroupCreateMkStream(ctx, stream, group, start)
}
func (i *instance) XGroupSetID(ctx context.Context, stream, group, start string) *rdsDrv.StatusCmd {
	return i.GetProxy().XGroupSetID(ctx, stream, group, start)
}
func (i *instance) XGroupDestroy(ctx context.Context, stream, group string) *rdsDrv.IntCmd {
	return i.GetProxy().XGroupDestroy(ctx, stream, group)
}
func (i *instance) XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *rdsDrv.IntCmd {
	return i.GetProxy().XGroupCreateConsumer(ctx, stream, group, consumer)
}
func (i *instance) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *rdsDrv.IntCmd {
	return i.GetProxy().XGroupDelConsumer(ctx, stream, group, consumer)
}
func (i *instance) XReadGroup(ctx context.Context, a *rdsDrv.XReadGroupArgs) *rdsDrv.XStreamSliceCmd {
	return i.GetProxy().XReadGroup(ctx, a)
}
func (i *instance) XAck(ctx context.Context, stream, group string, ids ...string) *rdsDrv.IntCmd {
	return i.GetProxy().XAck(ctx, stream, group, ids...)
}
func (i *instance) XPending(ctx context.Context, stream, group string) *rdsDrv.XPendingCmd {
	return i.GetProxy().XPending(ctx, stream, group)
}
func (i *instance) XPendingExt(ctx context.Context, a *rdsDrv.XPendingExtArgs) *rdsDrv.XPendingExtCmd {
	return i.GetProxy().XPendingExt(ctx, a)
}
func (i *instance) XClaim(ctx context.Context, a *rdsDrv.XClaimArgs) *rdsDrv.XMessageSliceCmd {
	return i.GetProxy().XClaim(ctx, a)
}
func (i *instance) XClaimJustID(ctx context.Context, a *rdsDrv.XClaimArgs) *rdsDrv.StringSliceCmd {
	return i.GetProxy().XClaimJustID(ctx, a)
}
func (i *instance) XAutoClaim(ctx context.Context, a *rdsDrv.XAutoClaimArgs) *rdsDrv.XAutoClaimCmd {
	return i.GetProxy().XAutoClaim(ctx, a)
}
func (i *instance) XAutoClaimJustID(ctx context.Context, a *rdsDrv.XAutoClaimArgs) *rdsDrv.XAutoClaimJustIDCmd {
	return i.GetProxy().XAutoClaimJustID(ctx, a)
}
func (i *instance) XTrimMaxLen(ctx context.Context, key string, maxLen int64) *rdsDrv.IntCmd {
	return i.GetProxy().XTrimMaxLen(ctx, key, maxLen)
}
func (i *instance) XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *rdsDrv.IntCmd {
	return i.GetProxy().XTrimMaxLenApprox(ctx, key, maxLen, limit)
}
func (i *instance) XTrimMinID(ctx context.Context, key string, minID string) *rdsDrv.IntCmd {
	return i.GetProxy().XTrimMinID(ctx, key, minID)
}
func (i *instance) XTrimMinIDApprox(ctx context.Context, key string, minID string, limit int64) *rdsDrv.IntCmd {
	return i.GetProxy().XTrimMinIDApprox(ctx, key, minID, limit)
}
func (i *instance) XInfoGroups(ctx context.Context, key string) *rdsDrv.XInfoGroupsCmd {
	return i.GetProxy().XInfoGroups(ctx, key)
}
func (i *instance) XInfoStream(ctx context.Context, key string) *rdsDrv.XInfoStreamCmd {
	return i.GetProxy().XInfoStream(ctx, key)
}
func (i *instance) XInfoStreamFull(ctx context.Context, key string, count int) *rdsDrv.XInfoStreamFullCmd {
	return i.GetProxy().XInfoStreamFull(ctx, key, count)
}
func (i *instance) XInfoConsumers(ctx context.Context, key string, group string) *rdsDrv.XInfoConsumersCmd {
	return i.GetProxy().XInfoConsumers(ctx, key, group)
}
func (i *instance) BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) *rdsDrv.ZWithKeyCmd {
	return i.GetProxy().BZPopMax(ctx, timeout, keys...)
}
func (i *instance) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) *rdsDrv.ZWithKeyCmd {
	return i.GetProxy().BZPopMin(ctx, timeout, keys...)
}
func (i *instance) BZMPop(ctx context.Context, timeout time.Duration, order string,
	count int64, keys ...string) *rdsDrv.ZSliceWithKeyCmd {
	return i.GetProxy().BZMPop(ctx, timeout, order, count, keys...)
}
func (i *instance) ZAdd(ctx context.Context, key string, members ...rdsDrv.Z) *rdsDrv.IntCmd {
	return i.GetProxy().ZAdd(ctx, key, members...)
}
func (i *instance) ZAddLT(ctx context.Context, key string, members ...rdsDrv.Z) *rdsDrv.IntCmd {
	return i.GetProxy().ZAddLT(ctx, key, members...)
}
func (i *instance) ZAddGT(ctx context.Context, key string, members ...rdsDrv.Z) *rdsDrv.IntCmd {
	return i.GetProxy().ZAddGT(ctx, key, members...)
}
func (i *instance) ZAddNX(ctx context.Context, key string, members ...rdsDrv.Z) *rdsDrv.IntCmd {
	return i.GetProxy().ZAddNX(ctx, key, members...)
}
func (i *instance) ZAddXX(ctx context.Context, key string, members ...rdsDrv.Z) *rdsDrv.IntCmd {
	return i.GetProxy().ZAddXX(ctx, key, members...)
}
func (i *instance) ZAddArgs(ctx context.Context, key string, args rdsDrv.ZAddArgs) *rdsDrv.IntCmd {
	return i.GetProxy().ZAddArgs(ctx, key, args)
}
func (i *instance) ZAddArgsIncr(ctx context.Context, key string, args rdsDrv.ZAddArgs) *rdsDrv.FloatCmd {
	return i.GetProxy().ZAddArgsIncr(ctx, key, args)
}
func (i *instance) ZCard(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().ZCard(ctx, key)
}
func (i *instance) ZCount(ctx context.Context, key, min, max string) *rdsDrv.IntCmd {
	return i.GetProxy().ZCount(ctx, key, min, max)
}
func (i *instance) ZLexCount(ctx context.Context, key, min, max string) *rdsDrv.IntCmd {
	return i.GetProxy().ZLexCount(ctx, key, min, max)
}
func (i *instance) ZIncrBy(ctx context.Context, key string, increment float64, member string) *rdsDrv.FloatCmd {
	return i.GetProxy().ZIncrBy(ctx, key, increment, member)
}
func (i *instance) ZInter(ctx context.Context, store *rdsDrv.ZStore) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZInter(ctx, store)
}
func (i *instance) ZInterWithScores(ctx context.Context, store *rdsDrv.ZStore) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZInterWithScores(ctx, store)
}
func (i *instance) ZInterCard(ctx context.Context, limit int64, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().ZInterCard(ctx, limit, keys...)
}
func (i *instance) ZInterStore(ctx context.Context, destination string, store *rdsDrv.ZStore) *rdsDrv.IntCmd {
	return i.GetProxy().ZInterStore(ctx, destination, store)
}
func (i *instance) ZMPop(ctx context.Context, order string, count int64, keys ...string) *rdsDrv.ZSliceWithKeyCmd {
	return i.GetProxy().ZMPop(ctx, order, count, keys...)
}
func (i *instance) ZMScore(ctx context.Context, key string, members ...string) *rdsDrv.FloatSliceCmd {
	return i.GetProxy().ZMScore(ctx, key, members...)
}
func (i *instance) ZPopMax(ctx context.Context, key string, count ...int64) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZPopMax(ctx, key, count...)
}
func (i *instance) ZPopMin(ctx context.Context, key string, count ...int64) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZPopMin(ctx, key, count...)
}
func (i *instance) ZRange(ctx context.Context, key string, start, stop int64) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRange(ctx, key, start, stop)
}
func (i *instance) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRangeWithScores(ctx, key, start, stop)
}
func (i *instance) ZRangeByScore(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRangeByScore(ctx, key, opt)
}
func (i *instance) ZRangeByLex(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRangeByLex(ctx, key, opt)
}
func (i *instance) ZRangeByScoreWithScores(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRangeByScoreWithScores(ctx, key, opt)
}
func (i *instance) ZRangeArgs(ctx context.Context, z rdsDrv.ZRangeArgs) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRangeArgs(ctx, z)
}
func (i *instance) ZRangeArgsWithScores(ctx context.Context, z rdsDrv.ZRangeArgs) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRangeArgsWithScores(ctx, z)
}
func (i *instance) ZRangeStore(ctx context.Context, dst string, z rdsDrv.ZRangeArgs) *rdsDrv.IntCmd {
	return i.GetProxy().ZRangeStore(ctx, dst, z)
}
func (i *instance) ZRank(ctx context.Context, key, member string) *rdsDrv.IntCmd {
	return i.GetProxy().ZRank(ctx, key, member)
}
func (i *instance) ZRankWithScore(ctx context.Context, key, member string) *rdsDrv.RankWithScoreCmd {
	return i.GetProxy().ZRankWithScore(ctx, key, member)
}

func (i *instance) ZRem(ctx context.Context, key string, members ...any) *rdsDrv.IntCmd {
	return i.GetProxy().ZRem(ctx, key, members...)
}
func (i *instance) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *rdsDrv.IntCmd {
	return i.GetProxy().ZRemRangeByRank(ctx, key, start, stop)
}
func (i *instance) ZRemRangeByScore(ctx context.Context, key, min, max string) *rdsDrv.IntCmd {
	return i.GetProxy().ZRemRangeByScore(ctx, key, min, max)
}
func (i *instance) ZRemRangeByLex(ctx context.Context, key, min, max string) *rdsDrv.IntCmd {
	return i.GetProxy().ZRemRangeByLex(ctx, key, min, max)
}
func (i *instance) ZRevRange(ctx context.Context, key string, start, stop int64) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRevRange(ctx, key, start, stop)
}
func (i *instance) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRevRangeWithScores(ctx, key, start, stop)
}
func (i *instance) ZRevRangeByScore(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRevRangeByScore(ctx, key, opt)
}
func (i *instance) ZRevRangeByLex(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRevRangeByLex(ctx, key, opt)
}
func (i *instance) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt *rdsDrv.ZRangeBy) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRevRangeByScoreWithScores(ctx, key, opt)
}
func (i *instance) ZRevRank(ctx context.Context, key, member string) *rdsDrv.IntCmd {
	return i.GetProxy().ZRevRank(ctx, key, member)
}
func (i *instance) ZRevRankWithScore(ctx context.Context, key, member string) *rdsDrv.RankWithScoreCmd {
	return i.GetProxy().ZRevRankWithScore(ctx, key, member)
}
func (i *instance) ZScore(ctx context.Context, key, member string) *rdsDrv.FloatCmd {
	return i.GetProxy().ZScore(ctx, key, member)
}
func (i *instance) ZUnionStore(ctx context.Context, dest string, store *rdsDrv.ZStore) *rdsDrv.IntCmd {
	return i.GetProxy().ZUnionStore(ctx, dest, store)
}
func (i *instance) ZRandMember(ctx context.Context, key string, count int) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZRandMember(ctx, key, count)
}
func (i *instance) ZRandMemberWithScores(ctx context.Context, key string, count int) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZRandMemberWithScores(ctx, key, count)
}
func (i *instance) ZUnion(ctx context.Context, store rdsDrv.ZStore) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZUnion(ctx, store)
}
func (i *instance) ZUnionWithScores(ctx context.Context, store rdsDrv.ZStore) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZUnionWithScores(ctx, store)
}
func (i *instance) ZDiff(ctx context.Context, keys ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ZDiff(ctx, keys...)
}
func (i *instance) ZDiffWithScores(ctx context.Context, keys ...string) *rdsDrv.ZSliceCmd {
	return i.GetProxy().ZDiffWithScores(ctx, keys...)
}
func (i *instance) ZDiffStore(ctx context.Context, destination string, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().ZDiffStore(ctx, destination, keys...)
}
func (i *instance) PFAdd(ctx context.Context, key string, els ...any) *rdsDrv.IntCmd {
	return i.GetProxy().PFAdd(ctx, key, els...)
}
func (i *instance) PFCount(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().PFCount(ctx, keys...)
}
func (i *instance) PFMerge(ctx context.Context, dest string, keys ...string) *rdsDrv.StatusCmd {
	return i.GetProxy().PFMerge(ctx, dest, keys...)
}
func (i *instance) BgRewriteAOF(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().BgRewriteAOF(ctx)
}
func (i *instance) BgSave(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().BgSave(ctx)
}
func (i *instance) ClientKill(ctx context.Context, ipPort string) *rdsDrv.StatusCmd {
	return i.GetProxy().ClientKill(ctx, ipPort)
}
func (i *instance) ClientKillByFilter(ctx context.Context, keys ...string) *rdsDrv.IntCmd {
	return i.GetProxy().ClientKillByFilter(ctx, keys...)
}
func (i *instance) ClientList(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().ClientList(ctx)
}
func (i *instance) ClientPause(ctx context.Context, dur time.Duration) *rdsDrv.BoolCmd {
	return i.GetProxy().ClientPause(ctx, dur)
}
func (i *instance) ClientUnpause(ctx context.Context) *rdsDrv.BoolCmd {
	return i.GetProxy().ClientUnpause(ctx)
}
func (i *instance) ClientID(ctx context.Context) *rdsDrv.IntCmd {
	return i.GetProxy().ClientID(ctx)
}
func (i *instance) ClientUnblock(ctx context.Context, id int64) *rdsDrv.IntCmd {
	return i.GetProxy().ClientUnblock(ctx, id)
}
func (i *instance) ClientUnblockWithError(ctx context.Context, id int64) *rdsDrv.IntCmd {
	return i.GetProxy().ClientUnblockWithError(ctx, id)
}
func (i *instance) ClientInfo(ctx context.Context) *rdsDrv.ClientInfoCmd {
	return i.GetProxy().ClientInfo(ctx)
}
func (i *instance) ConfigGet(ctx context.Context, parameter string) *rdsDrv.MapStringStringCmd {
	return i.GetProxy().ConfigGet(ctx, parameter)
}
func (i *instance) ConfigResetStat(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ConfigResetStat(ctx)
}
func (i *instance) ConfigSet(ctx context.Context, parameter, value string) *rdsDrv.StatusCmd {
	return i.GetProxy().ConfigSet(ctx, parameter, value)
}
func (i *instance) ConfigRewrite(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ConfigRewrite(ctx)
}
func (i *instance) DBSize(ctx context.Context) *rdsDrv.IntCmd {
	return i.GetProxy().DBSize(ctx)
}
func (i *instance) FlushAll(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().FlushAll(ctx)
}
func (i *instance) FlushAllAsync(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().FlushAllAsync(ctx)
}
func (i *instance) FlushDB(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().FlushDB(ctx)
}
func (i *instance) FlushDBAsync(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().FlushDBAsync(ctx)
}
func (i *instance) Info(ctx context.Context, section ...string) *rdsDrv.StringCmd {
	return i.GetProxy().Info(ctx, section...)
}
func (i *instance) LastSave(ctx context.Context) *rdsDrv.IntCmd {
	return i.GetProxy().LastSave(ctx)
}
func (i *instance) Save(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().Save(ctx)
}
func (i *instance) Shutdown(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().Shutdown(ctx)
}
func (i *instance) ShutdownSave(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ShutdownSave(ctx)
}
func (i *instance) ShutdownNoSave(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ShutdownNoSave(ctx)
}
func (i *instance) SlaveOf(ctx context.Context, host, port string) *rdsDrv.StatusCmd {
	return i.GetProxy().SlaveOf(ctx, host, port)
}
func (i *instance) SlowLogGet(ctx context.Context, num int64) *rdsDrv.SlowLogCmd {
	return i.GetProxy().SlowLogGet(ctx, num)
}
func (i *instance) Time(ctx context.Context) *rdsDrv.TimeCmd { return i.GetProxy().Time(ctx) }
func (i *instance) DebugObject(ctx context.Context, key string) *rdsDrv.StringCmd {
	return i.GetProxy().DebugObject(ctx, key)
}
func (i *instance) ReadOnly(ctx context.Context) *rdsDrv.StatusCmd { return i.GetProxy().ReadOnly(ctx) }
func (i *instance) ReadWrite(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ReadWrite(ctx)
}
func (i *instance) MemoryUsage(ctx context.Context, key string, samples ...int) *rdsDrv.IntCmd {
	return i.GetProxy().MemoryUsage(ctx, key, samples...)
}
func (i *instance) Eval(ctx context.Context, script string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().Eval(ctx, script, keys, args...)
}
func (i *instance) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().EvalSha(ctx, sha1, keys, args...)
}
func (i *instance) EvalRO(ctx context.Context, script string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().EvalRO(ctx, script, keys, args...)
}
func (i *instance) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().EvalShaRO(ctx, sha1, keys, args...)
}
func (i *instance) ScriptExists(ctx context.Context, hashes ...string) *rdsDrv.BoolSliceCmd {
	return i.GetProxy().ScriptExists(ctx, hashes...)
}
func (i *instance) ScriptFlush(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ScriptFlush(ctx)
}
func (i *instance) ScriptKill(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ScriptKill(ctx)
}
func (i *instance) ScriptLoad(ctx context.Context, script string) *rdsDrv.StringCmd {
	return i.GetProxy().ScriptLoad(ctx, script)
}
func (i *instance) FunctionLoad(ctx context.Context, code string) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionLoad(ctx, code)
}
func (i *instance) FunctionLoadReplace(ctx context.Context, code string) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionLoadReplace(ctx, code)
}
func (i *instance) FunctionDelete(ctx context.Context, libName string) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionDelete(ctx, libName)
}
func (i *instance) FunctionFlush(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionFlush(ctx)
}
func (i *instance) FunctionKill(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionKill(ctx)
}
func (i *instance) FunctionFlushAsync(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionFlushAsync(ctx)
}
func (i *instance) FunctionList(ctx context.Context, q rdsDrv.FunctionListQuery) *rdsDrv.FunctionListCmd {
	return i.GetProxy().FunctionList(ctx, q)
}
func (i *instance) FunctionDump(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionDump(ctx)
}
func (i *instance) FunctionRestore(ctx context.Context, libDump string) *rdsDrv.StringCmd {
	return i.GetProxy().FunctionRestore(ctx, libDump)
}
func (i *instance) FunctionStats(ctx context.Context) *rdsDrv.FunctionStatsCmd {
	return i.GetProxy().FunctionStats(ctx)
}
func (i *instance) FCall(ctx context.Context, function string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().FCall(ctx, function, keys, args...)
}
func (i *instance) FCallRo(ctx context.Context, function string, keys []string, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().FCallRo(ctx, function, keys, args...)
}
func (i *instance) FCallRO(ctx context.Context, function string, keys []string, args ...interface{}) *rdsDrv.Cmd {
	return i.GetProxy().FCallRO(ctx, function, keys, args...)
}
func (i *instance) Publish(ctx context.Context, channel string, message any) *rdsDrv.IntCmd {
	return i.GetProxy().Publish(ctx, channel, message)
}
func (i *instance) SPublish(ctx context.Context, channel string, message any) *rdsDrv.IntCmd {
	return i.GetProxy().SPublish(ctx, channel, message)
}
func (i *instance) PubSubChannels(ctx context.Context, pattern string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().PubSubChannels(ctx, pattern)
}
func (i *instance) PubSubNumSub(ctx context.Context, channels ...string) *rdsDrv.MapStringIntCmd {
	return i.GetProxy().PubSubNumSub(ctx, channels...)
}
func (i *instance) PubSubNumPat(ctx context.Context) *rdsDrv.IntCmd {
	return i.GetProxy().PubSubNumPat(ctx)
}
func (i *instance) PubSubShardChannels(ctx context.Context, pattern string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().PubSubShardChannels(ctx, pattern)
}
func (i *instance) PubSubShardNumSub(ctx context.Context, channels ...string) *rdsDrv.MapStringIntCmd {
	return i.GetProxy().PubSubShardNumSub(ctx, channels...)
}
func (i *instance) ClusterMyShardID(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().ClusterMyShardID(ctx)
}
func (i *instance) ClusterSlots(ctx context.Context) *rdsDrv.ClusterSlotsCmd {
	return i.GetProxy().ClusterSlots(ctx)
}
func (i *instance) ClusterShards(ctx context.Context) *rdsDrv.ClusterShardsCmd {
	return i.GetProxy().ClusterShards(ctx)
}
func (i *instance) ClusterLinks(ctx context.Context) *rdsDrv.ClusterLinksCmd {
	return i.GetProxy().ClusterLinks(ctx)
}
func (i *instance) ClusterNodes(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().ClusterNodes(ctx)
}
func (i *instance) ClusterMeet(ctx context.Context, host, port string) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterMeet(ctx, host, port)
}
func (i *instance) ClusterForget(ctx context.Context, nodeID string) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterForget(ctx, nodeID)
}
func (i *instance) ClusterReplicate(ctx context.Context, nodeID string) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterReplicate(ctx, nodeID)
}
func (i *instance) ClusterResetSoft(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterResetSoft(ctx)
}
func (i *instance) ClusterResetHard(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterResetHard(ctx)
}
func (i *instance) ClusterInfo(ctx context.Context) *rdsDrv.StringCmd {
	return i.GetProxy().ClusterInfo(ctx)
}
func (i *instance) ClusterKeySlot(ctx context.Context, key string) *rdsDrv.IntCmd {
	return i.GetProxy().ClusterKeySlot(ctx, key)
}
func (i *instance) ClusterGetKeysInSlot(ctx context.Context, slot int, count int) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ClusterGetKeysInSlot(ctx, slot, count)
}
func (i *instance) ClusterCountFailureReports(ctx context.Context, nodeID string) *rdsDrv.IntCmd {
	return i.GetProxy().ClusterCountFailureReports(ctx, nodeID)
}
func (i *instance) ClusterCountKeysInSlot(ctx context.Context, slot int) *rdsDrv.IntCmd {
	return i.GetProxy().ClusterCountKeysInSlot(ctx, slot)
}
func (i *instance) ClusterDelSlots(ctx context.Context, slots ...int) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterDelSlots(ctx, slots...)
}
func (i *instance) ClusterDelSlotsRange(ctx context.Context, min, max int) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterDelSlotsRange(ctx, min, max)
}
func (i *instance) ClusterSaveConfig(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterSaveConfig(ctx)
}
func (i *instance) ClusterSlaves(ctx context.Context, nodeID string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().ClusterSlaves(ctx, nodeID)
}
func (i *instance) ClusterFailover(ctx context.Context) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterFailover(ctx)
}
func (i *instance) ClusterAddSlots(ctx context.Context, slots ...int) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterAddSlots(ctx, slots...)
}
func (i *instance) ClusterAddSlotsRange(ctx context.Context, min, max int) *rdsDrv.StatusCmd {
	return i.GetProxy().ClusterAddSlotsRange(ctx, min, max)
}
func (i *instance) GeoAdd(ctx context.Context, key string, geoLocation ...*rdsDrv.GeoLocation) *rdsDrv.IntCmd {
	return i.GetProxy().GeoAdd(ctx, key, geoLocation...)
}
func (i *instance) GeoPos(ctx context.Context, key string, members ...string) *rdsDrv.GeoPosCmd {
	return i.GetProxy().GeoPos(ctx, key, members...)
}
func (i *instance) GeoRadius(ctx context.Context, key string, longitude, latitude float64,
	query *rdsDrv.GeoRadiusQuery) *rdsDrv.GeoLocationCmd {
	return i.GetProxy().GeoRadius(ctx, key, longitude, latitude, query)
}
func (i *instance) GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64,
	query *rdsDrv.GeoRadiusQuery) *rdsDrv.IntCmd {
	return i.GetProxy().GeoRadiusStore(ctx, key, longitude, latitude, query)
}
func (i *instance) GeoRadiusByMember(ctx context.Context, key, member string,
	query *rdsDrv.GeoRadiusQuery) *rdsDrv.GeoLocationCmd {
	return i.GetProxy().GeoRadiusByMember(ctx, key, member, query)
}
func (i *instance) GeoRadiusByMemberStore(ctx context.Context, key, member string,
	query *rdsDrv.GeoRadiusQuery) *rdsDrv.IntCmd {
	return i.GetProxy().GeoRadiusByMemberStore(ctx, key, member, query)
}
func (i *instance) GeoSearch(ctx context.Context, key string, q *rdsDrv.GeoSearchQuery) *rdsDrv.StringSliceCmd {
	return i.GetProxy().GeoSearch(ctx, key, q)
}
func (i *instance) GeoSearchLocation(ctx context.Context, key string,
	q *rdsDrv.GeoSearchLocationQuery) *rdsDrv.GeoSearchLocationCmd {
	return i.GetProxy().GeoSearchLocation(ctx, key, q)
}
func (i *instance) GeoSearchStore(ctx context.Context, key, store string,
	q *rdsDrv.GeoSearchStoreQuery) *rdsDrv.IntCmd {
	return i.GetProxy().GeoSearchStore(ctx, key, store, q)
}
func (i *instance) GeoDist(ctx context.Context, key string, member1, member2, unit string) *rdsDrv.FloatCmd {
	return i.GetProxy().GeoDist(ctx, key, member1, member2, unit)
}
func (i *instance) GeoHash(ctx context.Context, key string, members ...string) *rdsDrv.StringSliceCmd {
	return i.GetProxy().GeoHash(ctx, key, members...)
}
func (i *instance) ACLDryRun(ctx context.Context, username string, command ...any) *rdsDrv.StringCmd {
	return i.GetProxy().ACLDryRun(ctx, username, command...)
}
func (i *instance) ModuleLoadex(ctx context.Context, conf *rdsDrv.ModuleLoadexConfig) *rdsDrv.StringCmd {
	return i.GetProxy().ModuleLoadex(ctx, conf)
}
func (i *instance) AddHook(hk rdsDrv.Hook) {
	i.GetProxy().AddHook(hk)
}
func (i *instance) Watch(ctx context.Context, fn func(*rdsDrv.Tx) error, keys ...string) error {
	return i.GetProxy().Watch(ctx, fn, keys...)
}
func (i *instance) Do(ctx context.Context, args ...any) *rdsDrv.Cmd {
	return i.GetProxy().Do(ctx, args...)
}
func (i *instance) Process(ctx context.Context, cmd rdsDrv.Cmder) error {
	return i.GetProxy().Process(ctx, cmd)
}
func (i *instance) Subscribe(ctx context.Context, channels ...string) *rdsDrv.PubSub {
	return i.GetProxy().Subscribe(ctx, channels...)
}
func (i *instance) PSubscribe(ctx context.Context, channels ...string) *rdsDrv.PubSub {
	return i.GetProxy().PSubscribe(ctx, channels...)
}
func (i *instance) SSubscribe(ctx context.Context, channels ...string) *rdsDrv.PubSub {
	return i.GetProxy().SSubscribe(ctx, channels...)
}
func (i *instance) Close() error {
	return i.GetProxy().Close()
}
func (i *instance) PoolStats() *rdsDrv.PoolStats {
	return i.GetProxy().PoolStats()
}
