export function fmtNumber(v, decimals = 2) {
    if (v == null || Number.isNaN(Number(v))) return '-';
    return Number(v).toLocaleString(undefined, { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

export function fmtTime(v) {
    if (!v) return '-';
    const d = new Date(v);
    if (Number.isNaN(d.getTime())) return '-';
    return d.toLocaleTimeString();
}

export function fmtDate(v) {
    if (!v) return '-';
    const d = new Date(v);
    if (Number.isNaN(d.getTime())) return '-';
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
}
