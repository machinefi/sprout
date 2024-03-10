// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

library JournalParser {
    uint8 constant OPENING_SQUARE = 91;
    uint8 constant CLOSING_SQUARE = 93;
    uint8 constant COMMA = 44;
    uint8 constant ASCII_NUM_START = 48;
    uint8 constant OPENING_CURLY = 123;
    uint8 constant CLOSING_CURLY = 125;
    uint8 constant QUOTES = 34;
    uint8 constant COLUMN = 58;

    struct Device {
        uint256 id;
        uint256 reward;
    }

    function byteStringToBytes(bytes memory proof_snark_journal) internal pure returns (bytes memory) {
        bytes memory byteValues = new bytes(proof_snark_journal.length);

        uint256 byteValuesIndex = 0;
        uint256 start = 0;

        for (uint256 i = 0; i < proof_snark_journal.length; i++) {
            uint8 char = uint8(proof_snark_journal[i]);

            if (char == OPENING_SQUARE) {
                start++;
                continue;
            }

            if (char == COMMA || char == CLOSING_SQUARE) {
                uint8 digitsInByte = uint8(i - start);

                uint8 byteValue = 0;

                for (uint j = 0; j < digitsInByte; j++) {
                    uint exp = digitsInByte - j;
                    uint8 asciiValue = uint8(proof_snark_journal[start]);
                    uint8 digit = asciiValue - ASCII_NUM_START;
                    byteValue += uint8(digit * (10 ** (exp - 1)));
                    ++start;
                }

                start++;

                byteValues[byteValuesIndex] = bytes1(byteValue);
                byteValuesIndex++;
            }
        }

        return byteValues;
    }

    function parseDeviceJson(bytes memory jsonString) internal pure returns (Device[] memory, uint256) {
        Device[] memory devices = new Device[](jsonString.length);

        uint256 deviceIndex = 0;
        uint256 pointer = 0;

        bool quoteOpened = false;

        Device memory device;

        for (uint256 i = 0; i < jsonString.length; i++) {
            uint8 char = uint8(jsonString[i]);

            if (char == OPENING_CURLY) {
                pointer = i + 1;
                continue;
            }

            if (char == QUOTES && !quoteOpened) {
                quoteOpened = true;
                pointer = i + 1;
                continue;
            }

            if (char == COLUMN) {
                pointer++;
                continue;
            }

            if (char == QUOTES && quoteOpened) {
                quoteOpened = false;
                uint256 digitsInId = i - pointer;
                uint256 idValue = 0;

                for (uint j = 0; j < digitsInId; j++) {
                    uint256 exp = digitsInId - j;
                    uint8 asciiValue = uint8(jsonString[pointer]);
                    uint8 digit = asciiValue - ASCII_NUM_START;
                    idValue += uint8(digit * (10 ** (exp - 1)));
                    pointer++;
                }

                pointer++;
                device.id = idValue;
            }

            if (char == COMMA || char == CLOSING_CURLY) {
                uint256 digitsInReward = i - pointer;
                uint256 rewardValue = 0;

                for (uint j = 0; j < digitsInReward; j++) {
                    uint256 exp = digitsInReward - j;
                    uint8 asciiValue = uint8(jsonString[pointer]);
                    uint8 digit = asciiValue - ASCII_NUM_START;
                    rewardValue += uint8(digit * (10 ** (exp - 1)));
                    pointer++;
                }

                pointer++;
                device.reward = rewardValue;

                devices[deviceIndex] = Device(device.id, device.reward);
                deviceIndex++;
            }
        }

        return (devices, deviceIndex);
    }
}
