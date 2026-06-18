import { ReactNode } from 'react';

type FieldProps = {
  label: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
};

export function Field({ label, hint, error, required, children }: FieldProps) {
  return (
    <label className={error ? 'field field-invalid' : 'field'}>
      <span className="field-label">
        {label}
        {required ? <span className="required-mark">Required</span> : null}
      </span>
      {children}
      {error ? <span className="field-error">{error}</span> : null}
      {hint ? <span className="field-hint">{hint}</span> : null}
    </label>
  );
}
