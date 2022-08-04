import TextField from "@mui/material/TextField/TextField";

export const VolumeInput = ({
  volumes,
  value,
  onChange,
  hasError,
  setHasError,
}) => {
  const handleChange = (newName) => {
    const exists = volumes.some((volume) => volume.volumeName === newName);
    setHasError(exists);
    onChange(newName);
  };

  return (
    <TextField
      fullWidth
      label="Volume name"
      value={value}
      error={hasError}
      onChange={(event) => handleChange(event.target.value)}
      helperText={
        hasError
          ? "This name is already in use"
          : "Leave this field empty if you want an automatically generated name."
      }
    />
  );
};
