// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract ProjectRegistrar is ERC721, ReentrancyGuard {
    struct Project {
        string uri;
        bytes32 hash;
        bool paused;
    }
    event AttributeSet(uint64 indexed projectId, bytes32 indexed key, bytes value);
    event OperatorAdded(uint64 indexed projectId, address indexed operator);
    event OperatorRemoved(uint64 indexed projectId, address indexed operator);
    event ProjectPaused(uint64 indexed projectId);
    event ProjectUnpaused(uint64 indexed projectId);
    event ProjectUpserted(uint64 indexed projectId, string uri, bytes32 hash);
    event RegistrationFeeSet(uint256 fee);

    uint64 private nextProjectId;
    uint256 private registrationFee;

    mapping(uint64 => Project) public projects;
    mapping(uint64 => mapping(address => bool)) public operators;
    mapping(uint64 => mapping(bytes32 => bytes)) public attributes;

    modifier onlyProjectOperator(uint64 _projectId) {
        require(canOperateProject(msg.sender, _projectId), "not operator");
        _;
    }

    modifier onlyProjectOwner(uint64 _projectId) {
        require(ownerOf(_projectId) == msg.sender, "not owner");
        _;
    }

    // TODO: make contract upgradable & reset registration fee
    constructor(uint256 _fee) ERC721("ProjectToken", "PTK") {
        nextProjectId = 1;
        registrationFee = _fee;
        emit RegistrationFeeSet(_fee);
    }

    function SetAttributes(uint64 _projectId, bytes32[] memory _keys, bytes[] memory _values) public onlyProjectOperator(_projectId) {
        require(_keys.length == _values.length, "invalid input");
        mapping(bytes32 => bytes) storage project = attributes[_projectId];
        for (uint i = 0; i < _keys.length; i++) {
            project[_keys[i]] = _values[i];
            emit AttributeSet(_projectId, _keys[i], _values[i]);
        }
    }

    function AttributesOf(uint64 _projectId, bytes32[] memory _keys) public view returns (bytes[] memory values_) {
        require(ownerOf(_projectId) != address(0), "invalid project id");
        values_ = new bytes[](_keys.length);
        mapping(bytes32 => bytes) storage project = attributes[_projectId];
        for (uint i = 0; i < _keys.length; i++) {
            values_[i] = project[_keys[i]];
        }
    }

    function canOperateProject(address _operator, uint64 _projectId) public view returns (bool) {
        return ownerOf(_projectId) == _operator || operators[_projectId][_operator];
    }

    function createProject(string memory _uri, bytes32 _hash) public nonReentrant payable {
        require(msg.value >= registrationFee, "insufficient value");
        uint64 projectId = nextProjectId++;
        _mint(msg.sender, projectId);
        updateProjectInternal(projectId, _uri, _hash);
    }

    function addOperator(uint64 _projectId, address _operator) public onlyProjectOwner(_projectId) {
        operators[_projectId][_operator] = true;
        emit OperatorAdded(_projectId, _operator);
    }

    function removeOperator(uint64 _projectId, address _operator) public onlyProjectOwner(_projectId) {
        operators[_projectId][_operator] = false;
        emit OperatorRemoved(_projectId, _operator);
    }

    function pauseProject(uint64 _projectId) public onlyProjectOperator(_projectId) {
        Project storage project = projects[_projectId];
        require(!project.paused, "Project is already paused");
        project.paused = true;
        emit ProjectPaused(_projectId);
    }

    function unpauseProject(uint64 _projectId) public onlyProjectOperator(_projectId) {
        Project storage project = projects[_projectId];
        require(project.paused, "Project is not paused");
        project.paused = false;
        emit ProjectUnpaused(_projectId);
    }

    function updateProject(uint64 _projectId, string memory _uri, bytes32 _hash) public onlyProjectOperator(_projectId) {
        updateProjectInternal(_projectId, _uri, _hash);
    }

    function updateProjectInternal(uint64 _projectId, string memory _uri, bytes32 _hash) private {
        require(bytes(_uri).length != 0, "Empty uri value");
        Project storage project = projects[_projectId];
        project.uri = _uri;
        project.hash = _hash;
        emit ProjectUpserted(_projectId, _uri, _hash);
    }
}
