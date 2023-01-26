from pyteal import *

global_service_name = Bytes("service_name")
global_last_sn = Bytes("last_sn")

is_creator = Txn.sender() == Global.creator_address()

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
    return output.set(Substring(address.get(), Int(6), getSlashPosition(address)))

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
    "bsh-handler",
    BareCallActions(
        no_op=OnCompleteAction.create_only(
            Seq(
                App.globalPut(global_service_name, Bytes("bts")),
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
def sendServiceMessage(bmc_app: abi.Application, to: abi.String) -> Expr:
    net_to = abi.make(abi.String)
    dst_account = abi.make(abi.String)
    sn = abi.make(abi.Uint64)

    return Seq(
        checkPrefix(to),
        extractNetworkLabel(to).store_into(net_to),
        extractAppLabel(to).store_into(dst_account),
        getNextSn().store_into(sn),
        
        (service_name := abi.String()).set(App.globalGet(global_service_name)),
        (msg := abi.String()).set("hello world"),
        
        InnerTxnBuilder.Begin(),
        InnerTxnBuilder.MethodCall(
            app_id=bmc_app.application_id(),
            method_signature="sendMessage(string,string,uint64,byte[])void",
            args=[net_to, service_name, sn, msg.encode()],
            extra_fields={
                TxnField.fee: Int(0)
            }
        ),
        InnerTxnBuilder.Submit(),
    )

@router.method
def handleBTPMessage(msg: abi.String) -> Expr:
    return Seq(
        Log(msg.get()),
    )

