// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {IERC721} from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import {Initializable} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

import {IFleetManager} from "./interfaces/IFleetManager.sol";
import {IWSRouter} from "./interfaces/IRouter.sol";
import {IWSReceiver} from "./interfaces/IReceiver.sol";

contract W3bstreamRouter is IWSRouter, Initializable {
    address public override owner;
    address public override admin;
    address public override projectRegistry;
    address public override fleetManager;

    function initialize(address _projectRegistry, address _fleetManager) public initializer {
        owner = msg.sender;
        admin = msg.sender;
        projectRegistry = _projectRegistry;
        fleetManager = _fleetManager;
    }

    function submit(
        uint256 _projectId,
        address _receiver,
        bytes32 _batchMR,
        bytes32 _devicesMR,
        bytes calldata _zkProof
    ) external {
        if (!IFleetManager(fleetManager).isAllowed(msg.sender, _projectId)) {
            revert NotOperator();
        }

        try IWSReceiver(_receiver).receiveData(_batchMR, _devicesMR, _zkProof) {
            emit DataReceived(msg.sender, true, "");
        } catch Error(string memory revertReason) {
            emit DataReceived(msg.sender, false, revertReason);
        }
    }

    function setFleetManager(address _fleetManager) external override {
        if (msg.sender != admin) {
            revert NotAdmin();
        }
        if (_fleetManager == address(0)) revert ZeroAddress();

        fleetManager = _fleetManager;
        emit FleetManagerChanged(_fleetManager);
    }

    function setOwner(address _owner) external override {
        if (msg.sender != owner) {
            revert NotOwner();
        }
        if (_owner == address(0)) revert ZeroAddress();

        owner = _owner;
        emit OwnerChanged(_owner);
    }

    function setAdmin(address _admin) external override {
        if (msg.sender != owner) {
            revert NotOwner();
        }
        if (_admin == address(0)) revert ZeroAddress();

        admin = _admin;
        emit AdminChanged(_admin);
    }

    function setProjectRegistry(address _projectRegistry) external override {
        if (msg.sender != admin) {
            revert NotAdmin();
        }
        if (_projectRegistry == address(0)) revert ZeroAddress();

        projectRegistry = _projectRegistry;
        emit ProjectRegistryChanged(_projectRegistry);
    }
}
