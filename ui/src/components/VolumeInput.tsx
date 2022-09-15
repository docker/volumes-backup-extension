import TextField from "@mui/material/TextField/TextField";

interface Props {
  volumes?: Array<{ volumeName: string }>;
  value: string;
  onChange(v: string): void;
  hasError: boolean;
  setHasError(v: boolean): void;
}

export const VolumeInput = ({
  volumes,
  value,
  onChange,
  hasError,
  setHasError,
}: Props) => {
  const handleChange = (newName: string) => {
    const exists = volumes.some(
      (volume) => volume.volumeName.toLowerCase() === newName.toLowerCase()
    );
    setHasError(exists);
    onChange(newName);
  };

  return (
    <TextField
      autoFocus
      fullWidth
      label="Volume name"
      value={value}
      error={hasError}
      onChange={(event) => handleChange(event.target.value)}
      onBlur={(event) => handleChange(event.target.value)}
      helperText={
        hasError
          ? "This name is already in use"
          : "Leave this field empty if you want an automatically generated name."
      }
    />
  );
};
