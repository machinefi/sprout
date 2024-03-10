// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {IERC721} from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import {IWSReceiver} from "../interfaces/IWSReceiver.sol";
import {IDeviceRegistry} from "../interfaces/IDeviceRegistry.sol";
import {IDeviceReward} from "../interfaces/IDeviceReward.sol";

import "../lib/JournalParser.sol";

contract WSReceiver is IWSReceiver {
    address public deviceNFTRegistry;
    address public tokenAddress;

    uint256 currentBatchHeight;
    mapping(uint256 => Batch) batches;

    constructor(address _deviceNFTRegistry, address _tokenAddress) {
        deviceNFTRegistry = _deviceNFTRegistry;
        tokenAddress = _tokenAddress;
    }

    error DeviceIsNotEnabled(uint256);

    function receiveData(bytes calldata _data) external {
        (
            bytes memory proof_snark_seal,
            bytes memory proof_snark_post_state_digest,
            bytes memory proof_snark_journal
        ) = abi.decode(_data, (bytes, bytes, bytes));

        bytes memory devicesJsonString = JournalParser.byteStringToBytes(proof_snark_journal);

        (JournalParser.Device[] memory devices, uint256 devicesLen) = JournalParser.parseDeviceJson(devicesJsonString);

        for (uint256 i; i < devicesLen; i++) {
            uint256 deviceId = devices[i].id;
            IDeviceRegistry.Device memory device = IDeviceRegistry(deviceNFTRegistry).getDeviceInfo(deviceId);
            if (!device.enabled) {
                revert DeviceIsNotEnabled(devices[i].id);
            }

            address owner = IERC721(deviceNFTRegistry).ownerOf(deviceId);
            IDeviceReward(tokenAddress).mint(owner, devices[i].reward);
        }

        currentBatchHeight++;
    }

    function getBatchHeight() external view returns (uint256) {
        return currentBatchHeight;
    }

    function _verify(bytes calldata) internal pure returns (bool) {
        return true;
    }
}
