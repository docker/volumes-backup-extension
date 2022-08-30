import { TextField } from "@mui/material";
import { useGetImages } from "../hooks/useGetImages";

interface Props {
  value: string;
  hasError: boolean
  setHasError(v: boolean): void;
  onChange(v: string): void;
}

export const NewImageInput = ({ value, hasError, setHasError, onChange }: Props) => {
  const { data: images } = useGetImages();
  const imageNames: string[] = (images || [])
    .reduce((acc, image) => {
      const namesWithoutTags = (image.RepoTags || []).map(tag => tag.split(":")[0]);
      acc = acc.concat(namesWithoutTags);
      return acc;
    }, [])
    .filter((name) => !name.includes("none"));

  const handleChange = (newValue: string) => {
    if (imageNames.some(name => name.toLowerCase() === newValue.toLowerCase())) {
      setHasError(true);
      onChange(newValue);
    } else {
      setHasError(false);
      onChange(newValue);
    }
  };
  return (
    <TextField
      fullWidth
      label="New image"
      value={value}
      error={hasError}
      helperText={hasError ? "This name is already in use" : ""}
      onChange={(e) => handleChange(e.target.value)}
    />
  );
};
