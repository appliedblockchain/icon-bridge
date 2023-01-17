from pyteal import *

MAX_DATA_SIZE = Int(2048)
MAX_ROLLBACK_SIZE = Int(1024)

is_creator = Txn.sender() == Global.creator_address()
is_called_by_contract = Global.caller_app_id() == Int(0)

global_protocol_fee = Bytes("protocol_fee")
global_last_sn = Bytes("last_sn")

class CallRequest(abi.NamedTuple):
    address_from: abi.Field[abi.Address]
    to: abi.Field[abi.String]
    rollback: abi.Field[abi.DynamicArray[abi.Byte]]
    enabled: abi.Field[abi.Bool]

@Subroutine(TealType.none)
def checkPrefix(address: abi.String):
    return Assert(Substring(address.get(), Int(0), Int(6)) == Bytes("btp://"), comment="PrefixIsNotSupported")

@Subroutine(TealType.uint64)
def getSlashPosition(address: abi.String):
    i = ScratchVar(TealType.uint64)
    position_slash = ScratchVar(TealType.uint64)

    return Seq([
        position_slash.store(Int(0)),
        For(i.store(Int(7)), i.load() < address.length(), i.store(i.load() + Int(1))).Do(
            If(Extract(address.get(), i.load(), Int(1)) == Bytes("/"))
            .Then(
                position_slash.store(i.load()),
                Break()
            )
        ),
        Return(position_slash.load())
    ])

@ABIReturnSubroutine
def extractNetworkLabel(
    address: abi.String, *, output: abi.String
) -> Expr:
    return output.set(Substring(address.get(), Int(7), getSlashPosition(address)))

@ABIReturnSubroutine
def extractAppLabel(
    address: abi.String, *, output: abi.String
) -> Expr:
    return output.set(Substring(address.get(), getSlashPosition(address) + Int(1), address.length()))


@ABIReturnSubroutine
def getNextSn(
    *, output: abi.Uint64
) -> Expr:
    return Seq([
        App.globalPut(global_last_sn, Itob(Btoi(App.globalGet(global_last_sn)) + Int(1))),
        output.set(Btoi(App.globalGet(global_last_sn)))
    ])

router = Router(
    "xcall-handler",
    BareCallActions(
        no_op=OnCompleteAction.create_only(
            Seq(
                App.globalPut(global_last_sn, Itob(Int(1))),
                Approve()
            )
        ),
        update_application=OnCompleteAction.always(Return(is_creator)),
        delete_application=OnCompleteAction.always(Return(is_creator)),
        clear_state=OnCompleteAction.never(),
    ),
)

@router.method
def setProtocolFee(value: abi.Uint64): 
    return Seq(
        Assert(value.get() >= Int(0), comment="ValueShouldBePositive"),
        App.globalPut(global_protocol_fee, value.get()),
        Approve()
    )

@router.method
def sendCallMessage(to: abi.String, data: abi.DynamicArray[abi.Byte], rollback: abi.DynamicArray[abi.Byte]): 
    request = CallRequest()
    net_to = abi.make(abi.String)
    dst_account = abi.make(abi.String)
    sn = abi.make(abi.Uint64)
    need_response = abi.make(abi.Bool)
    enabled = abi.make(abi.Bool)

    return Seq(
        Assert(is_called_by_contract, comment="SenderNotAContract"),
        Assert(data.length() <= MAX_DATA_SIZE, comment="MaxDataSizeExceeded"), 
        Assert(rollback.length() <= MAX_ROLLBACK_SIZE, comment="MaxRollbackSizeExceeded"),

        need_response.set(Int(0)),
        If(rollback.length() > Int(0), need_response.set(Int(1))),

        checkPrefix(to),
        extractNetworkLabel(to).store_into(net_to),
        extractAppLabel(to).store_into(dst_account),

        getNextSn().store_into(sn),
        (msg_sn := abi.Uint64()).set(Int(0)),
        
        If(need_response.get() == Int(1))
        .Then(
            (address_from:=abi.Address()).set(Txn.sender()),
            (enabled:=abi.Bool()).set(Int(0)),

            request.set(address_from, dst_account, rollback, enabled),

            App.box_put(Itob(sn.get()), request.encode()),
            
            msg_sn.set(sn.get()),
        ),
        
        Approve()
    )
