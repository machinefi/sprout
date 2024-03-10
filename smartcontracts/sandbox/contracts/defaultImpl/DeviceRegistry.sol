// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";

import {IDeviceRegistry} from "../interfaces/IDeviceRegistry.sol";

contract DeviceRegistry is IDeviceRegistry, ERC721, ERC721URIStorage, Ownable {
    mapping(uint256 => Device) private _devices;
    uint256 private _nextTokenId;

    event DeviceEnabled(uint256 indexed _deviceId);
    event DeviceDisabled(uint256 indexed _deviceId);

    constructor() ERC721("DeviceRegistry", "DRY") Ownable() {}

    modifier onlyDisabledDevice(uint256 _deviceId) {
        require(!_devices[_deviceId].enabled, "DeviceRegistry: device is already enabled");
        _;
    }

    modifier onlyEnabledDevice(uint256 _deviceId) {
        require(_devices[_deviceId].enabled, "DeviceRegistry: device is not enabled");
        _;
    }

    function safeMint(string memory _pubKey, address to, string memory uri) public onlyOwner {
        bytes32 pubKey = keccak256(abi.encodePacked(_pubKey));

        uint256 tokenId = _nextTokenId++;

        _safeMint(to, tokenId);
        _setTokenURI(tokenId, uri);

        _devices[tokenId] = Device({uri: uri, enabled: false, pubKey: pubKey});
    }

    function enable(uint256 _deviceId) public onlyOwner onlyDisabledDevice(_deviceId) {
        _devices[_deviceId].enabled = true;
        emit DeviceEnabled(_deviceId);
    }

    function disable(uint256 _deviceId) public onlyOwner onlyEnabledDevice(_deviceId) {
        _devices[_deviceId].enabled = false;
        emit DeviceDisabled(_deviceId);
    }

    function getDeviceInfo(uint256 _deviceId) public view override returns (Device memory) {
        return _devices[_deviceId];
    }

    // The following functions are overrides required by Solidity.

    function tokenURI(uint256 tokenId) public view override(ERC721, ERC721URIStorage) returns (string memory) {
        return super.tokenURI(tokenId);
    }

    function supportsInterface(bytes4 interfaceId) public view override(ERC721, ERC721URIStorage) returns (bool) {
        return super.supportsInterface(interfaceId);
    }

    function _burn(uint256 tokenId) internal override(ERC721, ERC721URIStorage) {
        super._burn(tokenId);
    }
}
