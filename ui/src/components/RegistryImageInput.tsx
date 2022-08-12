import TextField from "@mui/material/TextField/TextField";

interface Props {
  error: string;
  value: string;
  setValue(newValue: string): void;
  setError(newValue: string): void;
}
export const RegistryImageInput = ({
  error,
  value,
  setValue,
  setError,
}: Props) => {
  const handleRegistryImageValidation = (newVal: string) => {
    if (!newVal) setError(null);
    if (!new RegExp(/(?:.*\/)([^:]+)(?::.+)?/gm).test(newVal)) {
      setError("Please specify at least <user>/<repo-name>:<tag>.");
    } else {
      setError(null);
    }
  };

  return (
    <TextField
      fullWidth
      label="<registry>/<user>/<repo-name>:<tag>"
      helperText={
        error || "The default registry is DockerHub, if you do not specify one."
      }
      placeholder="docker.io/johndoe/my-image-name:latest"
      value={value}
      error={!!error}
      onChange={(e) => setValue(e.target.value)}
      onBlur={(e) => handleRegistryImageValidation(e.target.value)}
    />
  );
};
