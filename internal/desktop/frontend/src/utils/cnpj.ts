// compactCode keeps the letters and digits of a typed CNPJ or access key,
// uppercased, so a pasted "12.345.678/0001-00" or a key in groups of 4 match
// as the backend stores them. CNPJs, and so access keys, may carry letters
// (IN RFB 2.229/2024). Clearable inputs set null, which reads as ''.
export function compactCode(value: string | null) {
  return (value ?? '').replace(/[^0-9A-Za-z]/g, '').toUpperCase()
}

// normalizeCNPJInput turns what was typed or pasted in a CNPJ field into the
// 14 characters the backend stores. q-input may emit a number, read as text.
export function normalizeCNPJInput(value: string | number | null) {
  return compactCode(value === null ? null : String(value)).slice(0, 14)
}

const firstCheckDigitWeights = [5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2]
const secondCheckDigitWeights = [6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2]

// cnpjError returns why value is not a valid CNPJ, or '' when it is. It
// follows cnpj.Validate in internal/foundation/cnpj: numeric and
// alphanumeric CNPJs both pass, and the last two characters are numeric
// check digits.
export function cnpjError(value: string | null) {
  const cnpj = compactCode(value)
  if (cnpj === '') return 'Informe o CNPJ.'
  if (cnpj.length !== 14) return 'CNPJ deve ter 14 caracteres.'
  if (!/^[0-9]{2}$/.test(cnpj.slice(12))) {
    return 'Os dois últimos caracteres do CNPJ devem ser numéricos.'
  }
  if (!hasValidCheckDigits(cnpj)) return 'CNPJ inválido.'
  return ''
}

function hasValidCheckDigits(cnpj: string) {
  if (allSameCharacters(cnpj)) return false
  const base = cnpj.slice(0, 12)
  const first = calculateCheckDigit(base, firstCheckDigitWeights)
  const second = calculateCheckDigit(base + String(first), secondCheckDigitWeights)
  return cnpj.slice(12) === `${first}${second}`
}

function allSameCharacters(cnpj: string) {
  return [...cnpj].every((char) => char === cnpj[0])
}

// calculateCheckDigit applies the mod-11 rule over the character values set by
// the Receita: each character is worth its ASCII code minus 48, so '0'..'9'
// are worth 0..9 and 'A'..'Z' are worth 17..42.
function calculateCheckDigit(base: string, weights: number[]) {
  let sum = 0
  for (let i = 0; i < base.length; i++) {
    sum += (base.charCodeAt(i) - 48) * (weights[i] ?? 0)
  }
  const remainder = sum % 11
  return remainder < 2 ? 0 : 11 - remainder
}
