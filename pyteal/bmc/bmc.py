from pyteal import *

global_bsh_app_address = Bytes("bsh_app_address")
global_relayer_acc_address = Bytes("relayer_acc_address")
global_btp_address = Bytes("btp_address")
global_route = Bytes("route")

is_creator = Txn.sender() == Global.creator_address()
is_relayer = Txn.sender() == App.globalGet(global_relayer_acc_address)
is_bsh = Txn.sender() == App.globalGet(global_bsh_app_address)

class BMCMessage(abi.NamedTuple):
    Src: abi.Field[abi.String]
    Dst: abi.Field[abi.String]
    Svc: abi.Field[abi.String]
    Sn: abi.Field[abi.Uint64]
    Message: abi.Field[abi.DynamicBytes]

@Subroutine(TealType.none)
def checkPrefix(address: abi.String):
    return Assert(Substring(address.get(), Int(0), Int(6)) == Bytes("btp://"), comment="PrefixIsNotSupported")

@Subroutine(TealType.none)
def checkRouteNetwork(network: abi.String):
    return Seq(
        (route := abi.String()).set(App.globalGet(global_route)),
        Assert(Substring(route.get(), Int(6), Int(6) + Len(network.get())) == network.get(), comment="RouteNotFound")
    )

router = Router(
    "bmc-handler",
    BareCallActions(
        no_op=OnCompleteAction.create_only(
            Seq(
                App.globalPut(global_relayer_acc_address, Global.creator_address()),
                App.globalPut(global_route, Bytes("btp://0x1.icon/0x12333")),
                App.globalPut(global_btp_address, Bytes("btp://0x1.algo/0x12333")),
                Approve()
            )
        ),
        update_application=OnCompleteAction.always(Return(is_creator)),
        delete_application=OnCompleteAction.always(Return(is_creator)),
        clear_state=OnCompleteAction.never(),
    ),
)

@router.method
def setBTPAddress(network: abi.String): 
    """Set BTP address for BMC in Algorand network, ex: btp://1234.algo/0xabcd"""

    return Seq(
        App.globalPut(global_btp_address, Concat(Bytes("btp://"), network.encode(), Bytes("/"), Global.current_application_address())),
        Approve()
    )

@router.method
def setRoute(route: abi.String): 
    """Set BTP address for Icon BMC, ex: btp://0x1.icon/0xabcd"""

    return Seq(
        checkPrefix(route),
        App.globalPut(global_route, route.encode()),
        Approve()
    )

@router.method
def registerBSHContract(bsh_app_address: abi.Address): 
    return Seq(
        Assert(is_creator),
        App.globalPut(global_bsh_app_address, bsh_app_address.get()),
        Approve()
    )

@router.method
def setRelayer(relayer_account: abi.Address): 
    return Seq(
        Assert(is_relayer),
        App.globalPut(global_relayer_acc_address, relayer_account.get()),
        Approve()
    )
    
@router.method
def sendMessage (to: abi.String, svc: abi.String, sn: abi.Uint64, msg: abi.DynamicBytes) -> Expr:
    bmcMessage = BMCMessage()
    
    return Seq(
        Assert(is_bsh),
        checkRouteNetwork(to),

        (src := abi.String()).set(App.globalGet(global_btp_address)),
        (dst := abi.String()).set(App.globalGet(global_route)),

        bmcMessage.set(src, dst, svc, sn, msg),
        Log(bmcMessage.encode()),

        Approve()
    )

@router.method
def handleRelayMessage (bsh_app: abi.Application, msg: abi.String,  *, output: abi.String) -> Expr:
    return Seq(
        Assert(is_relayer),
        InnerTxnBuilder.Begin(),
        InnerTxnBuilder.MethodCall(
            app_id=bsh_app.application_id(),
            method_signature="handleBTPMessage(string)void",
            args=[msg],
            extra_fields={
                TxnField.fee: Int(0)
            }
        ),
        InnerTxnBuilder.Submit(),
        output.set("event:start handleBTPMessage")
    )