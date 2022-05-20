// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.8.0;
pragma abicoder v2;

import "@openzeppelin/contracts/token/ERC1155/utils/ERC1155Holder.sol";
import "./BSHPeripheryV1.sol";
import "./BSHCoreV1.sol";

contract AnotherHolder is ERC1155Holder {
    BSHPeripheryV1 private bshs;
    BSHCoreV1 private bshc;
    using String for string;

    function deposit() external payable {}

    function addBSHContract(address _bshs, address _bshc) external {
        bshs = BSHPeripheryV1(_bshs);
        bshc = BSHCoreV1(_bshc);
    }

    function setApprove(
        address _erc20,
        address _operator,
        uint256 _value
    ) external {
        IERC20Tradable(_erc20).approve(_operator, _value);
    }

    function callTransfer(
        string calldata _coinName,
        uint256 _value,
        string calldata _to
    ) external {
        bshc.transfer(_coinName, _value, _to);
    }

    // function isSendingNative(string[] memory _coinNames)
    //     private
    //     pure
    //     returns (int256)
    // {
    //     for (uint256 i = 0; i < _coinNames.length; i++) {
    //         if (_coinNames[i].compareTo("PARA")) {
    //             return int256(i);
    //         }
    //     }
    //     return -1;
    // }

    function callTransferBatch(
        address _bsh,
        string[] memory _coinNames,
        uint256[] memory _values,
        string calldata _to,
        uint256 _native
    ) external {
        // int256 pos = isSendingNative(_coinNames);
        if (_native != 0) {
            (bool success, bytes memory err) = _bsh.call{ value: _native }(
                abi.encodeWithSignature(
                    "transferBatch(string[],uint256[],string)",
                    _coinNames,
                    _values,
                    _to
                )
            );
            require(success, string(err));
        } else {
            bshc.transferBatch(_coinNames, _values, _to);
        }
    }
}
