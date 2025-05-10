#!/usr/bin/env bash

payloads=(
  '{"originationNumber":"+11111111111", "messageBody":"pray"}'
  '{"originationNumber":"+11111111111", "messageBody":"intercessor one"}'
  '{"originationNumber":"+11111111111", "messageBody":"2"}'
  '{"originationNumber":"+11111111111", "messageBody":"100"}'

  '{"originationNumber":"+12222222222", "messageBody":"pray"}'
  '{"originationNumber":"+12222222222", "messageBody":"intercessor two"}'
  '{"originationNumber":"+12222222222", "messageBody":"2"}'
  '{"originationNumber":"+12222222222", "messageBody":"100"}'

  '{"originationNumber":"+13333333333", "messageBody":"pray"}'
  '{"originationNumber":"+13333333333", "messageBody":"intercessor three"}'
  '{"originationNumber":"+13333333333", "messageBody":"2"}'
  '{"originationNumber":"+13333333333", "messageBody":"100"}'

  '{"originationNumber":"+14444444444", "messageBody":"pray"}'
  '{"originationNumber":"+14444444444", "messageBody":"intercessor four"}'
  '{"originationNumber":"+14444444444", "messageBody":"2"}'
  '{"originationNumber":"+14444444444", "messageBody":"100"}'

  '{"originationNumber":"+15555555555", "messageBody":"pray"}'
  '{"originationNumber":"+15555555555", "messageBody":"intercessor five"}'
  '{"originationNumber":"+15555555555", "messageBody":"2"}'
  '{"originationNumber":"+15555555555", "messageBody":"100"}'

  '{"originationNumber":"+16666666666", "messageBody":"pray"}'
  '{"originationNumber":"+16666666666", "messageBody":"intercessor six"}'
  '{"originationNumber":"+16666666666", "messageBody":"2"}'
  '{"originationNumber":"+16666666666", "messageBody":"100"}'

  '{"originationNumber":"+17777777777", "messageBody":"pray"}'
  '{"originationNumber":"+17777777777", "messageBody":"intercessor seven"}'
  '{"originationNumber":"+17777777777", "messageBody":"2"}'
  '{"originationNumber":"+17777777777", "messageBody":"100"}'

  '{"originationNumber":"+18888888888", "messageBody":"pray"}'
  '{"originationNumber":"+18888888888", "messageBody":"intercessor eight"}'
  '{"originationNumber":"+18888888888", "messageBody":"2"}'
  '{"originationNumber":"+18888888888", "messageBody":"100"}'

  '{"originationNumber":"+19999999999", "messageBody":"pray"}'
  '{"originationNumber":"+19999999999", "messageBody":"intercessor nine"}'
  '{"originationNumber":"+19999999999", "messageBody":"2"}'
  '{"originationNumber":"+19999999999", "messageBody":"100"}'

  '{"originationNumber":"+11234567890", "messageBody":"pray"}'
  '{"originationNumber":"+11234567890", "messageBody":"normal member one"}'
  '{"originationNumber":"+11234567890", "messageBody":"1"}'

  '{"originationNumber":"+12345678901", "messageBody":"pray"}'
  '{"originationNumber":"+12345678901", "messageBody":"normal member two"}'
  '{"originationNumber":"+12345678901", "messageBody":"1"}'

  '{"originationNumber":"+13456789012", "messageBody":"pray"}'
  '{"originationNumber":"+13456789012", "messageBody":"normal member three"}'
  '{"originationNumber":"+13456789012", "messageBody":"1"}'
)

for payload in "${payloads[@]}"; do
  echo "Sending: $payload"
  curl -s -X POST http://127.0.0.1:3000/ \
    -H 'Content-Type: application/json' \
    -d "$payload"
  echo ""
done