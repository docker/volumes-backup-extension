import TextField from "@mui/material/TextField/TextField";

interface Props {
  error: string;
  value: string;
  setValue(newValue: string): void;
  setError(newValue: string): void;
  volumeSize?: number;
}
export const RegistryImageInput = ({
  error,
  value,
  setValue,
  setError,
  volumeSize,
}: Props) => {
  const handleRegistryImageValidation = (newVal: string) => {
    if (!newVal) setError(null);
    // If volume exceeds 10GB, we prevent users from pushing it to DockerHub
    if(isDockerRegistry && sizeExceededMessage){
      setError("Pushing volumes larger than 10GB are not supported at the moment through this extension.");
      return;
    }
    if (!new RegExp(/(?:.*\/)([^:]+)(?::.+)?/gm).test(newVal)) {
      setError("Please specify at least <user>/<repo-name>:<tag>.");
    } else {
      setError(null);
    }
  };

  const isDockerRegistry = (value.split("/")[0].startsWith("docker.io") || value.split("/").length == 2);
  const sizeExceededMessage = volumeSize > 10 * 1000 * 1000 * 1000;

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
